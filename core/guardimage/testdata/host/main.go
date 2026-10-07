// SPDX-License-Identifier: LGPL-3.0-or-later
// Harmless runtime fixture for executable identity, not an admitted SAGE host.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/sage-x-project/sage-adk/core/tools"
)

func main() {
	fmt.Println("ready")
	input := bufio.NewScanner(os.Stdin)
	for input.Scan() {
		if input.Text() != "calculate" {
			return
		}
		result, err := tools.CalculatorTool().Execute(context.Background(), map[string]interface{}{"operation": "add", "a": float64(2), "b": float64(3)})
		if err != nil || json.NewEncoder(os.Stdout).Encode(result) != nil {
			return
		}
	}
}
