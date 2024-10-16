package main

import (
	"context"
	goflag "flag"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/containers/storage/pkg/reexec"
	"github.com/kubernetes-incubator/cri-o/lib"
	"github.com/kubernetes-incubator/cri-o/oci"
	"github.com/kubernetes-incubator/cri-o/pkg/signals"
	"github.com/kubernetes-incubator/cri-o/server"
	"github.com/kubernetes-incubator/cri-o/version"
	"github.com/projectatomic/libpod/pkg/hooks"
	_ "github.com/projectatomic/libpod/pkg/hooks/0.1.0"
	"github.com/sirupsen/logrus"
	"github.com/soheilhy/cmux"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
	runtime "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
)

// gitCommit is the commit that the binary is being built from.
// It will be populated by the Makefile.
var gitCommit = ""

func validateConfig(config *server.Config) error {
	switch config.ImageVolumes {
	case lib.ImageVolumesMkdir:
	case lib.ImageVolumesIgnore:
	case lib.ImageVolumesBind:
	default:
		return fmt.Errorf("Unrecognized image volume type specified")

	}

	if config.UIDMappings != "" && config.ManageNetworkNSLifecycle {
		return fmt.Errorf("Cannot use UIDMappings with ManageNetworkNSLifecycle")
	}
	if config.GIDMappings != "" && config.ManageNetworkNSLifecycle {
		return fmt.Errorf("Cannot use GIDMappings with ManageNetworkNSLifecycle")
	}

	if config.LogSizeMax >= 0 && config.LogSizeMax < oci.BufSize {
		return fmt.Errorf("log size max should be negative or >= %d", oci.BufSize)
	}
	return nil
}

func mergeConfig(config *server.Config, ctx *cli.Context) error {
	// Don't parse the config if the user explicitly set it to "".
	if path := ctx.GlobalString("config"); path != "" {
		if err := config.UpdateFromFile(path); err != nil {
			if ctx.GlobalIsSet("config") || !os.IsNotExist(err) {
				return err
			}

			// We don't error out if --config wasn't explicitly set and the
			// default doesn't exist. But we will log a warning about it, so
			// the user doesn't miss it.
			logrus.Warnf("default configuration file does not exist: %s", server.CrioConfigPath)
		}
	}

	// Override options set with the CLI.
	if ctx.GlobalIsSet("conmon") {
		config.Conmon = ctx.GlobalString("conmon")
	}
	if ctx.GlobalIsSet("pause-command") {
		config.PauseCommand = ctx.GlobalString("pause-command")
	}
	if ctx.GlobalIsSet("pause-image") {
		config.PauseImage = ctx.GlobalString("pause-image")
	}
	if ctx.GlobalIsSet("signature-policy") {
		config.SignaturePolicyPath = ctx.GlobalString("signature-policy")
	}
	if ctx.GlobalIsSet("root") {
		config.Root = ctx.GlobalString("root")
	}
	if ctx.GlobalIsSet("runroot") {
		config.RunRoot = ctx.GlobalString("runroot")
	}
	if ctx.GlobalIsSet("storage-driver") {
		config.Storage = ctx.GlobalString("storage-driver")
	}
	if ctx.GlobalIsSet("storage-opt") {
		config.StorageOptions = ctx.GlobalStringSlice("storage-opt")
	}
	if ctx.GlobalIsSet("file-locking") {
		config.FileLocking = ctx.GlobalBool("file-locking")
	}
	if ctx.GlobalIsSet("insecure-registry") {
		config.InsecureRegistries = ctx.GlobalStringSlice("insecure-registry")
	}
	if ctx.GlobalIsSet("registry") {
		config.Registries = ctx.GlobalStringSlice("registry")
	}
	if ctx.GlobalIsSet("default-transport") {
		config.DefaultTransport = ctx.GlobalString("default-transport")
	}
	if ctx.GlobalIsSet("listen") {
		config.Listen = ctx.GlobalString("listen")
	}
	if ctx.GlobalIsSet("stream-address") {
		config.StreamAddress = ctx.GlobalString("stream-address")
	}
	if ctx.GlobalIsSet("stream-port") {
		config.StreamPort = ctx.GlobalString("stream-port")
	}
	if ctx.GlobalIsSet("runtime") {
		config.Runtime = ctx.GlobalString("runtime")
	}
	if ctx.GlobalIsSet("selinux") {
		config.SELinux = ctx.GlobalBool("selinux")
	}
	if ctx.GlobalIsSet("seccomp-profile") {
		config.SeccompProfile = ctx.GlobalString("seccomp-profile")
	}
	if ctx.GlobalIsSet("apparmor-profile") {
		config.ApparmorProfile = ctx.GlobalString("apparmor-profile")
	}
	if ctx.GlobalIsSet("cgroup-manager") {
		config.CgroupManager = ctx.GlobalString("cgroup-manager")
	}
	if ctx.GlobalIsSet("hooks-dir-path") {
		config.HooksDirPath = ctx.GlobalString("hooks-dir-path")
	}
	if ctx.GlobalIsSet("default-mounts") {
		config.DefaultMounts = ctx.GlobalStringSlice("default-mounts")
	}
	if ctx.GlobalIsSet("default-mounts-file") {
		config.DefaultMountsFile = ctx.GlobalString("default-mounts-file")
	}
	if ctx.GlobalIsSet("default-capabilities") {
		config.DefaultCapabilities = strings.Split(ctx.GlobalString("default-capabilities"), ",")
	}
	if ctx.GlobalIsSet("pids-limit") {
		config.PidsLimit = ctx.GlobalInt64("pids-limit")
	}
	if ctx.GlobalIsSet("log-size-max") {
		config.LogSizeMax = ctx.GlobalInt64("log-size-max")
	}
	if ctx.GlobalIsSet("cni-config-dir") {
		config.NetworkDir = ctx.GlobalString("cni-config-dir")
	}
	if ctx.GlobalIsSet("cni-plugin-dir") {
		config.PluginDir = ctx.GlobalString("cni-plugin-dir")
	}
	if ctx.GlobalIsSet("image-volumes") {
		config.ImageVolumes = lib.ImageVolumesType(ctx.GlobalString("image-volumes"))
	}
	if ctx.GlobalIsSet("read-only") {
		config.ReadOnly = ctx.GlobalBool("read-only")
	}
	if ctx.GlobalIsSet("bind-mount-prefix") {
		config.BindMountPrefix = ctx.GlobalString("bind-mount-prefix")
	}
	if ctx.GlobalIsSet("uid-mappings") {
		config.UIDMappings = ctx.GlobalString("uid-mappings")
	}
	if ctx.GlobalIsSet("gid-mappings") {
		config.GIDMappings = ctx.GlobalString("gid-mappings")
	}
	return nil
}

func catchShutdown(ctx context.Context, cancel context.CancelFunc, gserver *grpc.Server, sserver *server.Server, hserver *http.Server, signalled *bool) {
	sig := make(chan os.Signal, 10)
	signal.Notify(sig, signals.Interrupt, signals.Term)
	go func() {
		for s := range sig {
			switch s {
			case signals.Interrupt:
				logrus.Debugf("Caught SIGINT")
			case signals.Term:
				logrus.Debugf("Caught SIGTERM")
			default:
				continue
			}
			*signalled = true
			gserver.GracefulStop()
			hserver.Shutdown(ctx)
			sserver.StopStreamServer()
			sserver.StopMonitors()
			cancel()
			if err := sserver.Shutdown(ctx); err != nil {
				logrus.Warnf("error shutting down main service %v", err)
			}
			return
		}
	}()
}

func main() {
	// https://github.com/kubernetes/kubernetes/issues/17162
	goflag.CommandLine.Parse([]string{})

	if reexec.Init() {
		return
	}
	app := cli.NewApp()

	var v []string
	v = append(v, version.Version)
	if gitCommit != "" {
		v = append(v, fmt.Sprintf("commit: %s", gitCommit))
	}
	app.Name = "crio"
	app.Usage = "crio server"
	app.Version = strings.Join(v, "\n")
	app.Metadata = map[string]interface{}{
		"config": server.DefaultConfig(),
	}

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config",
			Value: server.CrioConfigPath,
			Usage: "path to configuration file",
		},
		cli.StringFlag{
			Name:  "conmon",
			Usage: "path to the conmon executable",
		},
		cli.StringFlag{
			Name:  "listen",
			Usage: "path to crio socket",
		},
		cli.StringFlag{
			Name:  "stream-address",
			Usage: "bind address for streaming socket",
		},
		cli.StringFlag{
			Name:  "stream-port",
			Usage: "bind port for streaming socket (default: \"0\")",
		},
		cli.StringFlag{
			Name:  "log",
			Value: "",
			Usage: "set the log file path where internal debug information is written",
		},
		cli.StringFlag{
			Name:  "log-format",
			Value: "text",
			Usage: "set the format used by logs ('text' (default), or 'json')",
		},
		cli.StringFlag{
			Name:  "log-level",
			Usage: "log messages above specified level: debug, info (default), warn, error, fatal or panic",
		},

		cli.StringFlag{
			Name:  "pause-command",
			Usage: "name of the pause command in the pause image",
		},
		cli.StringFlag{
			Name:  "pause-image",
			Usage: "name of the pause image",
		},
		cli.StringFlag{
			Name:  "signature-policy",
			Usage: "path to signature policy file",
		},
		cli.StringFlag{
			Name:  "root",
			Usage: "crio root dir",
		},
		cli.StringFlag{
			Name:  "runroot",
			Usage: "crio state dir",
		},
		cli.StringFlag{
			Name:  "storage-driver",
			Usage: "storage driver",
		},
		cli.StringSliceFlag{
			Name:  "storage-opt",
			Usage: "storage driver option",
		},
		cli.BoolFlag{
			Name:  "file-locking",
			Usage: "enable or disable file-based locking",
		},
		cli.StringSliceFlag{
			Name:  "insecure-registry",
			Usage: "whether to disable TLS verification for the given registry",
		},
		cli.StringSliceFlag{
			Name:  "registry",
			Usage: "registry to be prepended when pulling unqualified images, can be specified multiple times",
		},
		cli.StringFlag{
			Name:  "default-transport",
			Usage: "default transport",
		},
		cli.StringFlag{
			Name:  "runtime",
			Usage: "OCI runtime path",
		},
		cli.StringFlag{
			Name:  "seccomp-profile",
			Usage: "default seccomp profile path",
		},
		cli.StringFlag{
			Name:  "apparmor-profile",
			Usage: "default apparmor profile name (default: \"crio-default\")",
		},
		cli.BoolFlag{
			Name:  "selinux",
			Usage: "enable selinux support",
		},
		cli.StringFlag{
			Name:  "cgroup-manager",
			Usage: "cgroup manager (cgroupfs or systemd)",
		},
		cli.Int64Flag{
			Name:  "pids-limit",
			Value: lib.DefaultPidsLimit,
			Usage: "maximum number of processes allowed in a container",
		},
		cli.Int64Flag{
			Name:  "log-size-max",
			Value: lib.DefaultLogSizeMax,
			Usage: "maximum log size in bytes for a container",
		},
		cli.StringFlag{
			Name:  "cni-config-dir",
			Usage: "CNI configuration files directory",
		},
		cli.StringFlag{
			Name:  "cni-plugin-dir",
			Usage: "CNI plugin binaries directory",
		},
		cli.StringFlag{
			Name:  "image-volumes",
			Value: string(lib.ImageVolumesMkdir),
			Usage: "image volume handling ('mkdir', 'bind', or 'ignore')",
		},
		cli.StringFlag{
			Name:   "hooks-dir-path",
			Usage:  "set the OCI hooks directory path",
			Value:  hooks.DefaultDir,
			Hidden: true,
		},
		cli.StringSliceFlag{
			Name:  "default-mounts",
			Usage: "add one or more default mount paths in the form host:container (deprecated)",
		},
		cli.StringFlag{
			Name:   "default-mounts-file",
			Usage:  "path to default mounts file",
			Hidden: true,
		},
		cli.StringFlag{
			Name:  "default-capabilities",
			Usage: "capabilities to add to the containers",
		},
		cli.BoolFlag{
			Name:  "profile",
			Usage: "enable pprof remote profiler on localhost:6060",
		},
		cli.IntFlag{
			Name:  "profile-port",
			Value: 6060,
			Usage: "port for the pprof profiler",
		},
		cli.BoolFlag{
			Name:  "enable-metrics",
			Usage: "enable metrics endpoint for the server on localhost:9090",
		},
		cli.IntFlag{
			Name:  "metrics-port",
			Value: 9090,
			Usage: "port for the metrics endpoint",
		},
		cli.BoolFlag{
			Name:  "read-only",
			Usage: "setup all unprivileged containers to run as read-only",
		},
		cli.StringFlag{
			Name:  "bind-mount-prefix",
			Usage: "specify a prefix to prepend to the source of a bind mount",
		},
		cli.StringFlag{
			Name:  "uid-mappings",
			Usage: "specify the UID mappings to use for the user namespace",
			Value: "",
		},
		cli.StringFlag{
			Name:  "gid-mappings",
			Usage: "specify the GID mappings to use for the user namespace",
			Value: "",
		},
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	sort.Sort(cli.FlagsByName(configCommand.Flags))

	app.Commands = []cli.Command{
		configCommand,
	}

	app.Before = func(c *cli.Context) error {
		// Load the configuration file.
		config := c.App.Metadata["config"].(*server.Config)
		if err := mergeConfig(config, c); err != nil {
			return err
		}

		if err := validateConfig(config); err != nil {
			return err
		}

		cf := &logrus.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05.000000000Z07:00",
			FullTimestamp:   true,
		}

		logrus.SetFormatter(cf)

		if loglevel := c.GlobalString("log-level"); loglevel != "" {
			level, err := logrus.ParseLevel(loglevel)
			if err != nil {
				return err
			}

			logrus.SetLevel(level)
		}

		if path := c.GlobalString("log"); path != "" {
			f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND|os.O_SYNC, 0666)
			if err != nil {
				return err
			}
			logrus.SetOutput(f)
		}

		switch c.GlobalString("log-format") {
		case "text":
			// retain logrus's default.
		case "json":
			logrus.SetFormatter(new(logrus.JSONFormatter))
		default:
			return fmt.Errorf("unknown log-format %q", c.GlobalString("log-format"))
		}

		return nil
	}

	app.Action = func(c *cli.Context) error {
		ctx, cancel := context.WithCancel(context.Background())

		if c.GlobalBool("profile") {
			profilePort := c.GlobalInt("profile-port")
			profileEndpoint := fmt.Sprintf("localhost:%v", profilePort)
			go func() {
				http.ListenAndServe(profileEndpoint, nil)
			}()
		}

		args := c.Args()
		if len(args) > 0 {
			for _, command := range app.Commands {
				if args[0] == command.Name {
					break
				}
			}
			return fmt.Errorf("command %q not supported", args[0])
		}

		config := c.App.Metadata["config"].(*server.Config)

		if !config.SELinux {
			disableSELinux()
		}

		if _, err := os.Stat(config.Runtime); os.IsNotExist(err) {
			// path to runtime does not exist
			return fmt.Errorf("invalid --runtime value %q", err)
		}

		if err := os.MkdirAll(filepath.Dir(config.Listen), 0755); err != nil {
			return err
		}

		// Remove the socket if it already exists
		if _, err := os.Stat(config.Listen); err == nil {
			if err := os.Remove(config.Listen); err != nil {
				logrus.Fatal(err)
			}
		}
		lis, err := server.Listen("unix", config.Listen)
		if err != nil {
			logrus.Fatalf("failed to listen: %v", err)
		}

		s := grpc.NewServer()

		service, err := server.New(ctx, config)
		if err != nil {
			logrus.Fatal(err)
		}

		if c.GlobalBool("enable-metrics") {
			metricsPort := c.GlobalInt("metrics-port")
			me, err := service.CreateMetricsEndpoint()
			if err != nil {
				logrus.Fatalf("Failed to create metrics endpoint: %v", err)
			}
			l, err := net.Listen("tcp", fmt.Sprintf(":%v", metricsPort))
			if err != nil {
				logrus.Fatalf("Failed to create listener for metrics: %v", err)
			}
			go func() {
				if err := http.Serve(l, me); err != nil {
					logrus.Fatalf("Failed to serve metrics endpoint: %v", err)
				}
			}()
		}

		runtime.RegisterRuntimeServiceServer(s, service)
		runtime.RegisterImageServiceServer(s, service)

		// after the daemon is done setting up we can notify systemd api
		notifySystem()

		go func() {
			service.StartExitMonitor()
		}()
		hookSync := make(chan error, 2)
		if service.ContainerServer.Hooks == nil {
			hookSync <- err // so we don't block during cleanup
		} else {
			go service.ContainerServer.Hooks.Monitor(ctx, hookSync)
			err = <-hookSync
			if err != nil {
				cancel()
				logrus.Fatal(err)
			}
		}

		m := cmux.New(lis)
		grpcL := m.Match(cmux.HTTP2HeaderField("content-type", "application/grpc"))
		httpL := m.Match(cmux.HTTP1Fast())

		infoMux := service.GetInfoMux()
		srv := &http.Server{
			Handler:     infoMux,
			ReadTimeout: 5 * time.Second,
		}

		graceful := false
		catchShutdown(ctx, cancel, s, service, srv, &graceful)

		go s.Serve(grpcL)
		go srv.Serve(httpL)

		serverCloseCh := make(chan struct{})
		go func() {
			defer close(serverCloseCh)
			if err := m.Serve(); err != nil {
				if graceful && strings.Contains(strings.ToLower(err.Error()), "use of closed network connection") {
					err = nil
				} else {
					logrus.Errorf("Failed to serve grpc request: %v", err)
				}
			}
		}()

		streamServerCloseCh := service.StreamingServerCloseChan()
		serverMonitorsCh := service.MonitorsCloseChan()
		select {
		case <-streamServerCloseCh:
		case <-serverMonitorsCh:
		case <-serverCloseCh:
		}

		service.Shutdown(ctx)
		cancel()

		<-streamServerCloseCh
		logrus.Debug("closed stream server")
		<-serverMonitorsCh
		logrus.Debug("closed monitors")
		err = <-hookSync
		if err == nil || err == context.Canceled {
			logrus.Debug("closed hook monitor")
		} else {
			logrus.Errorf("hook monitor failed: %v", err)
		}
		<-serverCloseCh
		logrus.Debug("closed main server")

		return nil
	}

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}
