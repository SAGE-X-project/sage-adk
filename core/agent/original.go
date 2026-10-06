// SPDX-License-Identifier: LGPL-3.0-or-later
package agent

import (
	"context"

	"github.com/sage-x-project/sage-adk/core/capture"
	"github.com/sage-x-project/sage-adk/pkg/types"
)

// InputDecoder interprets copies of the retained input after durable capture.
// It must be bounded, honor cancellation, and remain under trusted host control.
// Original bytes must come from the input boundary, not a re-serialized Message.
type InputDecoder func(context.Context, [][]byte) (*types.Message, error)

// ProcessOriginal persists input before decoding and rechecks it before invoking
// the agent. Only the trusted caller receives the Request capability. On a decode
// or processing failure a persisted Request may still be returned for audit.
// This entry point preserves original input; it does not mediate tool effects.
// Servers and legacy Process callers must opt into this boundary explicitly.
func ProcessOriginal(ctx context.Context, a Agent, host *capture.Host, inputs [][]byte, decode InputDecoder) (request *capture.Request, response *types.Message, err error) {
	defer func() {
		if recover() != nil {
			response = nil
			err = capture.ErrCapture
		}
	}()
	if ctx == nil || ctx.Err() != nil || a == nil || decode == nil {
		return nil, nil, capture.ErrCapture
	}
	request, err = host.Capture(ctx, inputs)
	if err != nil {
		return nil, nil, err
	}
	retained, err := request.Inputs(ctx)
	if err != nil {
		return request, nil, err
	}
	msg, err := decode(ctx, retained)
	if err != nil {
		return request, nil, err
	}
	if msg == nil || msg.Validate() != nil || ctx.Err() != nil {
		return request, nil, capture.ErrCapture
	}
	if _, err = request.Inputs(ctx); err != nil {
		return request, nil, err
	}
	response, err = a.Process(ctx, msg)
	return request, response, err
}
