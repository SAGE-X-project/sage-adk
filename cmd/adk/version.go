// Copyright (C) 2025 sage-x-project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

const (
	version   = "1.2.0"
	buildDate = "2025-10-10"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of SAGE ADK",
	Long:  `Display version information for the SAGE Agent Development Kit CLI.`,
	Run: func(cmd *cobra.Command, args []string) {
		verbose, _ := cmd.Flags().GetBool("verbose")

		if verbose {
			fmt.Printf("SAGE ADK CLI\n")
			fmt.Printf("Version:      %s\n", version)
			fmt.Printf("Build Date:   %s\n", buildDate)
			fmt.Printf("Go Version:   %s\n", runtime.Version())
			fmt.Printf("OS/Arch:      %s/%s\n", runtime.GOOS, runtime.GOARCH)
		} else {
			fmt.Printf("adk version %s\n", version)
		}
	},
}

func init() {
	versionCmd.Flags().BoolP("verbose", "v", false, "Show detailed version information")
}
