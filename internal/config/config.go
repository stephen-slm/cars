package config

import (
	"os"
	"runtime"
	"strings"
)

var currentEnvironment = ""

const DefaultEnvironment = "development"
const DevelopmentEnvironment = "development"

// GetCurrentEnvironment returns the current environment if the system
// is running in windows or a linux environment. E.g defaulting to
// linux for mac.
func GetCurrentEnvironment() string {
	if currentEnvironment != "" {
		return currentEnvironment
	}

	environment := os.Getenv("environment")

	if environment == "" {
		currentEnvironment = DefaultEnvironment
		return currentEnvironment
	}

	for _, s := range []string{"staging", "production", "development"} {
		if environment == s {
			currentEnvironment = s
			return currentEnvironment
		}
	}

	currentEnvironment = DefaultEnvironment
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
