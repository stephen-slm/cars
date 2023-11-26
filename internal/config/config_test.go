package config

import (
	"os"
	"sync"
	"testing"
)

func TestGetCurrentEnvironment(t *testing.T) {
	tests := []struct {
		name            string
		want            string
		environmentFlag string
	}{{
		name:            "should default if not provided",
		want:            DefaultEnvironment,
		environmentFlag: "",
	}, {
		name:            "should return staging if environment is set to staging",
		want:            "staging",
		environmentFlag: "staging",
	}, {
		name:            "should return staging if environment is set to production",
		want:            "production",
		environmentFlag: "production",
	}, {
		name:            "should return development if environment is set to development",
		want:            "development",
		environmentFlag: "development",
	}, {
		name:            "should default if value is defined but not production or staging",
		want:            DefaultEnvironment,
		environmentFlag: "invalid-value",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				os.Unsetenv("environment")
				currentEnvironment = ""
			}()

			envOnce = sync.Once{}
			_ = os.Setenv("environment", tt.environmentFlag)

			if got := GetCurrentEnvironment(); got != tt.want {
				t.Errorf("GetCurrentEnvironment() = %v, want %v", got, tt.want)
			}
		})
	}
}
