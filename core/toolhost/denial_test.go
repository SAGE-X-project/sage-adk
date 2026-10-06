// SPDX-License-Identifier: LGPL-3.0-or-later
package toolhost_test

import "testing"

// Scenario-controlled unit fixtures use only owned temporary files and inert
// arithmetic. No attack-capable client or bypass reproduction is generated.
func TestNativeWorkerDenialScenarios(t *testing.T) {
	for _, mode := range []string{"measurement", "policy", "panic", "error", "array", "invalid", "oversize", "cancel"} {
		t.Run(mode, func(t *testing.T) { runNativeFixture(t, mode) })
	}
}

func TestNativeTerminalClientRefusesHandoff(t *testing.T) { runNativeFixture(t, "terminal") }
