//go:build !linux || (!amd64 && !arm64)

// SPDX-License-Identifier: LGPL-3.0-or-later
package guardimage

import "context"

func start(context.Context, Config, []byte) (backend, error) { return nil, ErrUnsupported }
