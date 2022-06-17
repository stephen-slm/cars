package config

import (
	"os"
	"runtime"
	"strings"
)

var currentEnvironment = ""

const defaultEnvironment = "development"

// GetCurrentEnvironment returns the current environment if the system
// is running in windows or a linux environment. E.g defaulting to
// linux for mac.
func GetCurrentEnvironment() string {
	if currentEnvironment != "" {
		return currentEnvironment
	}

	environment := os.Getenv("environment")

	if environment == "" {
		currentEnvironment = defaultEnvironment
		return currentEnvironment
	}

	for _, s := range []string{"staging", "production"} {
		if environment == s {
			currentEnvironment = s
			return currentEnvironment
		}
	}

	currentEnvironment = defaultEnvironment
	return currentEnvironment

}

// GetCurrentOs returns the current environment if the system
// is running in windows or a linux environment. E.g defaulting to
// linux for mac.
func GetCurrentOs() string {
	if strings.ToLower(runtime.GOOS) == "windows" {
		return "windows"
	}

	return "linux"

}
