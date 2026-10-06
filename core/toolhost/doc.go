// SPDX-License-Identifier: LGPL-3.0-or-later

// Package toolhost connects pinned tool instances to the SAGE 0.10.0 native MCP
// owner. Only signed, policy-authorized, live admitted worker invocations reach
// tools; no unsigned execution route or raw tool getter is exposed. Host, provider
// and instance capabilities must remain isolated from model/plugin-controlled
// routes. Local wrappers and file hashes alone do not establish that isolation.
package toolhost
