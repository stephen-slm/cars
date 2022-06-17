package sandbox

import (
	"compile-and-run-sandbox/internal/memory"
)

type Runtime string

const (
	Default Runtime = ""
	GVisor  Runtime = "runsc"
)

type Profile struct {
	// The runtime the container image will be used. Please reference Runtime
	// for more information about which runtimes are currently supported.
	Runtime Runtime

	// If the container should be automatically removed at the end of execution.
	AutoRemove bool

	// The maximum amount of memory the container can use. If you set this
	// option, the minimum allowed value is 6m (6 megabytes). That is, you must
	// set the value to at least 6 megabytes.
	Memory memory.MemorySize

	// The amount of memory this container is allowed to swap to disk.
	MemorySwap memory.MemorySize
}

// ProfileValueMaps A map of profile ids to profiles, this mapping is used
// between the consumer to determine which profile to use.
var ProfileValueMaps = map[uint]*Profile{
	1: Profiles["production"],
}

// Profiles is a list of all currently supported profiles in the system
var Profiles = map[string]*Profile{
	"development_linux": {
		Runtime:    GVisor,
		AutoRemove: true,
		Memory:     memory.Gigabyte * 10,
	},
	"development_windows": {
		Runtime:    Default,
		AutoRemove: true,
		Memory:     memory.Gigabyte * 10,
	},
	"production": {
		Runtime:    GVisor,
		AutoRemove: true,
		Memory:     memory.Gigabyte,
	},
	"staging": {
		Runtime:    GVisor,
		AutoRemove: true,
		Memory:     memory.Gigabyte * 2,
	},
}
