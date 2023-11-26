package docker

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type dockerDaemonConfig struct {
	Runtimes map[string]struct {
		Path string `json:"path"`
	} `json:"runtimes"`
}

const GVisorRuntime = "runsc"

var installed bool
var once sync.Once

func IsGvisorInstalled() bool {
	once.Do(func() {
		defer func() {
			if installed {
				log.Warn().Str("runtime", GVisorRuntime).Msg("Docker Runtime")
			} else {
				log.Warn().Str("runtime", "default").Msg("Docker Runtime")
			}
		}()

		dockerDaemonPath := "/etc/docker/daemon.json"

		if _, err := os.Stat(dockerDaemonPath); errors.Is(err, os.ErrNotExist) {
			installed = false
			return
		}

		fileBytes, err := os.ReadFile(dockerDaemonPath)

		if err != nil {
			log.Err(err).Msg("failed to read daemon file but it exists")
			installed = false
			return
		}

		daemon := &dockerDaemonConfig{}
		_ = json.Unmarshal(fileBytes, &daemon)

		_, ok := daemon.Runtimes[GVisorRuntime]

		installed = ok
	})

	return installed
}
