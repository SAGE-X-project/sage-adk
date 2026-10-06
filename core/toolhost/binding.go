// SPDX-License-Identifier: LGPL-3.0-or-later
package toolhost

import "context"

// LoadedTool is one trusted immutable loaded instance. Check must verify its
// protected approved baseline, actual loaded code/dependencies and tool binding.
// Execute receives exact canonical JSON arguments without added defaults. It
// returns a JSON object only after its effect and dependent cleanup end. Both
// callbacks must be bounded, concurrent-safe and honor cancellation; a digest
// alone does not establish which code is loaded. Keep raw instances protected.
type LoadedTool interface {
	Check(context.Context, string, string) error
	Execute(context.Context, []byte) ([]byte, error)
}

// Binding is protected local configuration selecting one instance and its
// approved name and manifest digest. It is never constructed from peer input.
type Binding struct {
	Name           string
	ManifestDigest string
	Tool           LoadedTool
}
