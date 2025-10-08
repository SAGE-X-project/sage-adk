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

//go:build examples
// +build examples

package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/sage-x-project/sage-adk/adapters/sage"
)

func main() {
	// Parse command-line flags
	outputPath := flag.String("output", "./keys/agent.pem", "Output path for the private key")
	format := flag.String("format", "pem", "Key format: pem or jwk")
	showPublic := flag.Bool("show-public", false, "Display public key after generation")
	flag.Parse()

	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(*outputPath)
	if err := os.MkdirAll(outputDir, 0700); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Check if file already exists
	if _, err := os.Stat(*outputPath); err == nil {
		log.Printf("⚠️  Warning: File already exists at %s", *outputPath)
		fmt.Print("Overwrite? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			log.Println("Aborted.")
			return
		}
	}

	// Create KeyManager
	km := sage.NewKeyManager()

	// Generate Ed25519 key pair
	log.Println("🔑 Generating Ed25519 key pair...")
	keyPair, err := km.Generate()
	if err != nil {
		log.Fatalf("Failed to generate key: %v", err)
	}

	// Save key to file
	log.Printf("💾 Saving key to %s...", *outputPath)
	if *format == "jwk" {
		err = km.SaveToFileWithFormat(keyPair, *outputPath, "jwk")
	} else {
		err = km.SaveToFile(keyPair, *outputPath)
	}

	if err != nil {
		log.Fatalf("Failed to save key: %v", err)
	}

	// Set restrictive permissions
	if err := os.Chmod(*outputPath, 0600); err != nil {
		log.Printf("⚠️  Warning: Failed to set restrictive permissions: %v", err)
	}

	log.Println("✅ Key generated successfully!")
	log.Printf("📁 Location: %s", *outputPath)
	log.Printf("🔒 Format: %s", *format)
	log.Printf("🛡️  Permissions: 0600 (read/write owner only)")

	// Show public key if requested
	if *showPublic {
		publicKey, err := km.ExtractEd25519PublicKey(keyPair)
		if err != nil {
			log.Printf("❌ Failed to extract public key: %v", err)
			return
		}

		log.Println("\n📋 Public Key (base64):")
		fmt.Println(base64.StdEncoding.EncodeToString(publicKey))

		log.Println("\n📋 Public Key (hex):")
		fmt.Printf("%x\n", publicKey)
	}

	// Display security warning
	log.Println("\n⚠️  SECURITY WARNING:")
	log.Println("   - Never commit this key to version control")
	log.Println("   - Store securely (consider hardware wallet for production)")
	log.Println("   - Back up in secure location")
	log.Println("   - Use different keys for dev/staging/prod")

	// Display next steps
	log.Println("\n📝 Next Steps:")
	log.Println("   1. Register your agent's DID on the blockchain")
	log.Println("   2. Map your public key to your DID in the SAGE contract")
	log.Printf("   3. Set SAGE_PRIVATE_KEY_PATH=%s\n", *outputPath)
	log.Println("   4. Start your SAGE agent")
}
