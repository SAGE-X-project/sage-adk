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

// Package health provides health check endpoints for SAGE ADK agents.
//
// # Overview
//
// This package provides Kubernetes-compatible health check probes:
//   - Liveness: Is the agent running?
//   - Readiness: Can the agent serve requests?
//   - Startup: Has the agent finished initialization?
//
// # Liveness Probe
//
// Indicates if the agent is alive and should not be restarted:
//
//	liveness := health.NewLivenessChecker()
//	http.Handle("/health/live", health.Handler(liveness))
//
// Returns 200 if agent is running, 503 otherwise.
//
// # Readiness Probe
//
// Indicates if the agent is ready to serve traffic:
//
//	readiness := health.NewReadinessChecker(
//	    health.NewLLMHealthCheck(llmProvider),
//	    health.NewStorageHealthCheck(storage),
//	)
//	http.Handle("/health/ready", health.Handler(readiness))
//
// Checks all dependencies before marking ready.
//
// # Startup Probe
//
// Indicates if the agent has completed initialization:
//
//	startup := health.NewStartupChecker()
//	startup.MarkReady()  // Call when initialization complete
//	http.Handle("/health/startup", health.Handler(startup))
//
// Used for slow-starting agents to prevent premature restarts.
//
// # Custom Health Checks
//
// Implement the Checker interface for custom checks:
//
//	type CustomCheck struct{}
//
//	func (c *CustomCheck) Name() string {
//	    return "custom"
//	}
//
//	func (c *CustomCheck) Check(ctx context.Context) health.CheckResult {
//	    // Perform health check
//	    return health.CheckResult{
//	        Name:   c.Name(),
//	        Status: health.StatusHealthy,
//	    }
//	}
//
// # Kubernetes Integration
//
//	apiVersion: v1
//	kind: Pod
//	spec:
//	  containers:
//	  - name: agent
//	    livenessProbe:
//	      httpGet:
//	        path: /health/live
//	        port: 8080
//	      initialDelaySeconds: 30
//	      periodSeconds: 10
//	    readinessProbe:
//	      httpGet:
//	        path: /health/ready
//	        port: 8080
//	      initialDelaySeconds: 10
//	      periodSeconds: 5
//	    startupProbe:
//	      httpGet:
//	        path: /health/startup
//	        port: 8080
//	      failureThreshold: 30
//	      periodSeconds: 5
//
// # Response Format
//
// JSON response with health status:
//
//	{
//	  "name": "readiness",
//	  "status": "healthy",
//	  "details": {
//	    "checks": [
//	      {"name": "llm", "status": "healthy"},
//	      {"name": "storage", "status": "healthy"}
//	    ]
//	  }
//	}
package health
