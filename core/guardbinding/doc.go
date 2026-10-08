// SPDX-License-Identifier: LGPL-3.0-or-later
// Package guardbinding binds an approved exact-operation policy, protected
// root or actually admitted-hop capture and artifact snapshot to one trusted
// loaded instance. Open and OpenHop are separate explicit entry points. It is
// native host assembly, not OS isolation or deployment attestation. Keep every
// handle and provider outside model/plugin custody. Factory checks must attest
// the actual policy evaluator, tool and dependencies; file hashes alone cannot.
package guardbinding
