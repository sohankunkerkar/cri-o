package wasmartifact

import (
	"context"
	"fmt"

	"github.com/containers/image/v5/types"

	"github.com/cri-o/cri-o/internal/config/ociartifact"
	"github.com/cri-o/cri-o/internal/log"
	"github.com/cri-o/cri-o/pkg/annotations"
)

// WASMOCIArtifact is the main structure for handling WASM-related OCI artifacts.
type WASMOCIArtifact struct {
	impl Impl
}

// New creates a new WASM OCI artifact handler.
func New() *WASMOCIArtifact {
	return &WASMOCIArtifact{
		impl: ociartifact.New(),
	}
}

const (
	// WASMPodAnnotation is the annotation used for matching a whole pod
	// rather than a specific container.
	WASMPodAnnotation = annotations.WASMWorkloadAnnotation + "/POD"

	// requiredConfigMediaType is the config media type for OCI artifact WASM config.
	requiredConfigMediaType = "application/vnd.oci.image.config.v1+json"

	// requiredLayerMediaType is the layer media type for OCI artifact WASM workload.
	// requiredLayerMediaType = "application/vnd.wasm.content.layer.v1+wasm"
)

// TryPull tries to pull the OCI artifact WASM profile while evaluating
// the provided annotations.
func (w *WASMOCIArtifact) TryPull(
	ctx context.Context,
	sys *types.SystemContext,
	containerName, profileRef string,
) (data []byte, err error) {
	log.Debugf(ctx, "Evaluating WASM annotations")

	pullOptions := &ociartifact.PullOptions{
		SystemContext:          sys,
		EnforceConfigMediaType: requiredConfigMediaType,
		MaxSize:                5 * 1024 * 1024, // 5MB
	}
	artifact, err := w.impl.Pull(ctx, profileRef, pullOptions)
	if err != nil {
		return nil, fmt.Errorf("pull OCI artifact: %w", err)
	}

	log.Infof(ctx, "Retrieved OCI artifact WASM profile of len: %d", len(artifact.Data))
	return artifact.Data, nil
}
