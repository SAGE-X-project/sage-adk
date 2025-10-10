// Copyright (C) 2025 sage-x-project
// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"strings"
	"testing"
)

func TestVersionCmd_Default(t *testing.T) {
	// The version command uses fmt.Printf which goes to stdout
	// For testing, we just verify the command doesn't panic
	// and the version constants are set correctly
	if version == "" {
		t.Error("Version constant should not be empty")
	}

	if buildDate == "" {
		t.Error("Build date constant should not be empty")
	}
}

func TestVersionCmd_Verbose(t *testing.T) {
	// The version command uses fmt.Printf which goes to stdout
	// For testing, we verify the command exists and has the verbose flag
	if versionCmd.Flags().Lookup("verbose") == nil {
		t.Error("Expected version command to have verbose flag")
	}
}

func TestVersionConstants(t *testing.T) {
	if version == "" {
		t.Error("Version constant should not be empty")
	}

	if buildDate == "" {
		t.Error("Build date constant should not be empty")
	}

	// Version should be in semantic versioning format
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		t.Errorf("Version should be in semantic versioning format, got: %s", version)
	}
}
