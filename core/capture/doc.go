// SPDX-License-Identifier: LGPL-3.0-or-later

// Package capture preserves ordered original UTF-8 inputs at a trusted host
// boundary before decoding or model/plugin expansion. Its capabilities and stores
// must remain outside untrusted code. It binds retained inputs to Guard issuance;
// it does not mediate tool effects or attest the host's initial input capture.
package capture
