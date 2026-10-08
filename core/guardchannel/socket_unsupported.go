//go:build !linux || (!amd64 && !arm64)

// SPDX-License-Identifier: LGPL-3.0-or-later
package guardchannel

import (
	"context"
	image "github.com/sage-x-project/sage-adk/core/guardimage"
	"os"
)

func launch(context.Context, image.Config) (*image.Process, endpoint, error) {
	return nil, nil, image.ErrUnsupported
}
func child(file *os.File) (endpoint, error) { _ = file.Close(); return nil, image.ErrUnsupported }
