package docker

import (
	"encoding/json"
	"os"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type dockerDaemonConfig struct {
	Runtimes map[string]struct {
		Path string `json:"path"`
	} `json:"runtimes"`
}

const GVisorRuntime = "runsc"

var checked bool
var installed bool

func IsGvisorInstalled() bool {
	if checked {
		return installed
	}

	defer func() {
		if installed {
			log.Warn().Str("runtime", GVisorRuntime).Msg("Docker Runtime")
		} else {
			log.Warn().Str("runtime", "default").Msg("Docker Runtime")
		}
	}()

	checked = true
	dockerDaemonPath := "/etc/docker/daemon.json"

	if _, err := os.Stat(dockerDaemonPath); errors.Is(err, os.ErrNotExist) {
		installed = false
		return false
	}

	fileBytes, err := os.ReadFile(dockerDaemonPath)

	if err != nil {
		log.Err(err).Msg("failed to read daemon file but it exists")
		installed = false
		return false
	}

	daemon := &dockerDaemonConfig{}
	_ = json.Unmarshal(fileBytes, &daemon)

	_, ok := daemon.Runtimes[GVisorRuntime]

	installed = ok
	return ok
}
