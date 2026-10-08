// SPDX-License-Identifier: LGPL-3.0-or-later

// Package guardsigner keeps one Ed25519 signing key in a separate signer
// process. The protected host asks for signatures over a local Unix socket and
// never receives private key bytes. The server accepts only peers whose OS
// account is allowlisted and only messages that begin with an allowed SAGE
// 0.10.0 signing domain. The client checks that the socket is served by the
// expected signer account.
//
// This separates key bytes from the host process. It does not stop a
// compromised host from requesting signatures within its allowed domains, and
// it is not hardware-backed custody. The deployment still owns account
// separation, socket directory permissions, key file protection and isolation
// of both processes from model/plugin code. RFC 9421 HTTP signature bases have
// no domain prefix, so HTTP carriage is outside this signer's scope.
package guardsigner
