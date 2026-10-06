// SPDX-License-Identifier: LGPL-3.0-or-later
package capture

import (
	"context"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
)

// OpenMCPClient reopens an existing issued Client journal under the authenticated
// native owner. First close the old Client successfully and keep its exact path,
// signed intent and permanent issuance fence in protected custody. The host
// serializes this close-and-reopen transfer; it is not atomic across processes.
// Missing history is refused, never recreated. Connection capabilities are trusted
// host-only. The fresh capture check remains bound to later Client handoffs.
func (r *Request) OpenMCPClient(ctx context.Context, connection *g.MCPConnection, path string, intent []byte, s g.MCPClientServices) error {
	inputs, err := r.Inputs(ctx)
	if err != nil {
		return err
	}
	if connection == nil || s.IntentAuthority == nil || s.ResultAuthority == nil || absent(s.Policy) || absent(s.Clock) {
		return ErrCapture
	}
	root, err := g.NewRootCapture(inputs, r.id)
	if err != nil {
		return ErrCapture
	}
	s.Policy = &boundPolicy{request: r, delegate: s.Policy}
	return connection.OpenRootClient(path, false, append([]byte(nil), intent...), s, root)
}
