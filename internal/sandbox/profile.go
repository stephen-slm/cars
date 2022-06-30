package sandbox

import (
	"fmt"
	"time"

	"compile-and-run-sandbox/internal/config"
	"compile-and-run-sandbox/internal/docker"
	"compile-and-run-sandbox/internal/memory"
)

type Runtime string

const (
	Default Runtime = ""
	GVisor  Runtime = "runsc"
)

func (r Runtime) String() string {
	return string(r)
}

type Profile struct {
	// The runtime the container image will be used. Please reference Runtime
	// for more information about which runtimes are currently supported.
	Runtime Runtime
	// If the container should be automatically removed at the end of execution.
	AutoRemove bool
	// The max amount of timeout for the given executed code, if the code docker
	// container is running for longer than the given timeout then the code is
	// rejected.
	CodeTimeout time.Duration
	// The max amount of timeout for the given compile code, if the compiling
	// is running for longer than the given timeout then the code is rejected.
	CompileTimeout time.Duration
	// The maximum amount of memory the container can use. If you set this
	// option, the minimum allowed value is 6m (6 megabytes). That is, you must
	// set the value to at least 6 megabytes.
	ContainerMemory memory.Memory
	// The maximum amount of memory allowed to be used by the code execution.
	// Ideally keep the container memory higher to allow the code to fully use
	// its full range.
	ExecutionMemory memory.Memory
	// The amount of memory this container is allowed to swap to disk.
	MemorySwap memory.Memory
}

// ProfileValueMaps A map of profile ids to profiles, this mapping is used
// between the consumer to determine which profile to use.
var _ = map[uint]*Profile{
	1: profiles["production"],
}

// Profiles is a list of all currently supported profiles in the system
var profiles = map[string]*Profile{
	"development_linux": {
		AutoRemove:      true,
		CodeTimeout:     time.Second * 5,
		CompileTimeout:  time.Second * 20,
		ContainerMemory: memory.Gigabyte * 2,
		ExecutionMemory: memory.Gigabyte,
		Runtime:         GVisor,
	},
	"development_windows": {
		AutoRemove:      true,
		CodeTimeout:     time.Second * 10,
		CompileTimeout:  time.Second * 20,
		ContainerMemory: memory.Gigabyte * 2,
		ExecutionMemory: memory.Gigabyte,
		Runtime:         Default,
	},
	"production": {
		AutoRemove:      true,
		CodeTimeout:     time.Second,
		CompileTimeout:  time.Second * 10,
		ContainerMemory: memory.Gigabyte * 2,
		ExecutionMemory: memory.Gigabyte,
		Runtime:         GVisor,
	},
	"staging": {
		AutoRemove:      true,
		CodeTimeout:     time.Second * 2,
		CompileTimeout:  time.Second * 20,
		ContainerMemory: memory.Gigabyte * 2,
		ExecutionMemory: memory.Gigabyte,
		Runtime:         GVisor,
	},
}

// disableGVisorCheckWrapper checks to see if GVisor is installed and if it is
// not installed then the runtime will be reset to the default.
func disableGVisorCheckWrapper(profile *Profile) *Profile {
	if !docker.IsGvisorInstalled() {
		profile.Runtime = Default
	}

	return profile
}

// GetProfileForMachine gets the current execution profile based on the
// machine values, this is the operating system and environment the machine
// is running on.
func GetProfileForMachine() *Profile {
	currentEnv := config.GetCurrentEnvironment()
	currentOs := config.GetCurrentOs()

	envProfile, envProfileExists := profiles[currentEnv]
	envOsProfile, envOsProfileExists := profiles[fmt.Sprintf("%s_%s", currentEnv, currentOs)]

	if !envOsProfileExists && !envProfileExists {
		profile := profiles[config.DefaultEnvironment]
		return disableGVisorCheckWrapper(profile)
	}

	if envOsProfileExists {
		return disableGVisorCheckWrapper(envOsProfile)
	}

	return disableGVisorCheckWrapper(envProfile)
}
