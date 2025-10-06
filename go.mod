module github.com/sage-x-project/sage-adk

go 1.24

// Utilities
require github.com/google/uuid v1.6.0

// Use local modules for development
replace (
	github.com/sage-x-project/sage => ../../sage
	github.com/sage-x-project/sage-a2a-go => ../../sage-a2a-go
)
