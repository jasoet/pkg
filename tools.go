//go:build tools

// This file declares dependencies on tools used during development.
// It won't be included in the final binary but ensures tools are
// tracked in go.mod and can be installed with 'go mod download'.

package tools

import (
	// Linting and static analysis
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	
	// Build automation
	_ "github.com/magefile/mage"
	
	// Testing and coverage
	_ "github.com/stretchr/testify"
	_ "gotest.tools/gotestsum"
	
	// Documentation generation
	_ "github.com/swaggo/swag/cmd/swag"
	
	// Database migrations (if using migrate instead of GORM auto-migrate)
	_ "github.com/golang-migrate/migrate/v4/cmd/migrate"
	
	// Mock generation for testing
	_ "github.com/golang/mock/mockgen"
	
	// Security scanning
	_ "github.com/securecodewarrior/gosec/v2/cmd/gosec"
	
	// Dependency scanning
	_ "github.com/sonatypecommunity/nancy"
	
	// Code generation
	_ "github.com/deepmap/oapi-codegen/cmd/oapi-codegen"
)