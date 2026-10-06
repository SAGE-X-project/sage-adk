// SPDX-License-Identifier: LGPL-3.0-or-later

// Package guardservices supplies common-clock and registry-bound Ed25519 signing
// adapters for protected hosts. Custody, authoritative Registry Source, policy,
// loaded instances, storage and OS isolation remain deployment responsibilities.
// Never expose these capabilities to model or plugin-controlled code.
package guardservices
