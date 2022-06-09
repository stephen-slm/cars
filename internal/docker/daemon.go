package docker

import (
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"os"
)

type dockerDaemonConfig struct {
	Runtimes map[string]struct {
		Path string `json:"path"`
	} `json:"runtimes"`
}

const GVisorRuntime = "runsc"

func IsGvisorInstalled() bool {
	dockerDaemonPath := "/etc/docker/daemon.json"

	if _, err := os.Stat(dockerDaemonPath); errors.Is(err, os.ErrNotExist) {
		return false
	}

	fileBytes, err := os.ReadFile(dockerDaemonPath)

	if err != nil {
		log.Err(err).Msg("failed to read daemon file but it exists")
		return false
	}

	daemon := &dockerDaemonConfig{}
	_ = json.Unmarshal(fileBytes, &daemon)

	_, ok := daemon.Runtimes[GVisorRuntime]
	return ok
}
