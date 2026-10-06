//go:build !linux && !darwin

// SPDX-License-Identifier: LGPL-3.0-or-later
package guardbinding

import "os"

func openArtifact(*os.Root, string) (*os.File, error) { return nil, ErrDenied }
