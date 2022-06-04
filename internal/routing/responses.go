package routing

import (
	"compile-and-run-sandbox/internal/sandbox"
)

type DirectCompileResponse struct {
	Output []string `json:"output"`

	Status     sandbox.ContainerStatus     `json:"status"`
	TestStatus sandbox.ContainerTestStatus `json:"test_status"`

	RuntimeMs     int64 `json:"runtime_ms"`
	CompileTimeMs int64 `json:"compile_time_ms"`
}

type QueueCompileResponse struct {
	ID string `json:"id"`
}
