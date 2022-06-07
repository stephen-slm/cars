//go:build tools
// +build tools

// Package tools records build-time dependencies that aren't used by the
// library itself, but are tracked by go mod and required to lint and
// build the project.
package build

import (
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/jstemmer/go-junit-report"
	_ "golang.org/x/tools/cmd/goimports"
	_ "golang.org/x/tools/cmd/stringer"
)
