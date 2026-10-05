// SPDX-License-Identifier: LGPL-3.0-or-later

package client

import "github.com/sage-x-project/sage-adk/pkg/types"

// MessageStream exchanges messages on a legacy bidirectional gRPC connection.
type MessageStream interface {
	Send(*types.Message) error
	Recv() (*types.Message, error)
	Close() error
}
