// SPDX-License-Identifier: LGPL-3.0-or-later

package a2a

import (
	"encoding/json"
	"testing"

	"github.com/sage-x-project/sage-adk/pkg/types"
	a2aprotocol "trpc.group/trpc-go/trpc-a2a-go/protocol"
)

func TestDecodedTextPart(t *testing.T) {
	original := types.NewMessage(types.MessageRoleUser, []types.Part{types.NewTextPart("serialized fixture")})
	wire, err := json.Marshal(convertMessageToA2A(original))
	if err != nil {
		t.Fatal(err)
	}
	var decoded a2aprotocol.Message
	if err := json.Unmarshal(wire, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Kind != a2aprotocol.KindMessage {
		t.Fatal("missing message kind")
	}
	restored := convertA2AMessageToSDK(&decoded)
	if len(restored.Parts) != 1 || restored.Parts[0].(*types.TextPart).Text != "serialized fixture" {
		t.Fatal("decoder pointer part lost text")
	}
}
