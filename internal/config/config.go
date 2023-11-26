package config

import (
	"os"
	"runtime"
	"strings"
	"sync"
)

var currentEnvironment = ""

const DefaultEnvironment = "development"
const DevelopmentEnvironment = "development"

// envOnce is used to ensure concurrent tests only pull the value once at startup. While it is
// mainly used for tests, it also ensures safely with the chance the value is overwritten during
// runtime.
var envOnce sync.Once

// GetCurrentEnvironment returns the current environment if the system
// is running in windows or a linux environment. E.g defaulting to
// linux for mac.
func GetCurrentEnvironment() string {
	envOnce.Do(func() {
		currentEnvironment = os.Getenv("environment")

		if currentEnvironment == "" {
			currentEnvironment = DefaultEnvironment
			return
		}

		for _, s := range []string{"staging", "production", "development"} {
			if currentEnvironment == s {
				currentEnvironment = s
				return
			}
		}

		currentEnvironment = DefaultEnvironment
	})

	return currentEnvironment
}

// GetCurrentOs returns the current environment if the system
// is running in windows or a linux environment. E.g defaulting to
// linux for mac.
func GetCurrentOs() string {
	if strings.EqualFold(runtime.GOOS, "windows") {
		return "windows"
	}

	return "linux"

}
