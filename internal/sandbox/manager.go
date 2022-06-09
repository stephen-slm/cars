package sandbox

import (
	"compile-and-run-sandbox/internal/docker"
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type ContainerManager struct {
	containers   sync.Map
	dockerClient *client.Client

	// If G-visor should be used as a container runtime for all container executions
	// or should it be using the default container runtime.
	enableGVisor bool

	// the limiter will be a buffered channel used to determine the number of possible
	// containers that can be executed at any one time. When the container is removed
	// the buffered channel will be popped, pushing will result in a block until the
	// pop has been executed.
	limiter  chan string
	stopFlag int32
}

func NewSandboxContainerManager(dockerClient *client.Client, maxConcurrentContainers int) *ContainerManager {
	manager := &ContainerManager{
		limiter:      make(chan string, maxConcurrentContainers),
		dockerClient: dockerClient,
		containers:   sync.Map{},
	}

	// check to see if we are configured to use g-visor or not.
	// if we are not configured then we should warn to the user
	// otherwise allow all containers  to use it.
	if docker.IsGvisorInstalled() {
		log.Warn().Str("runtime", docker.GVisorRuntime).Msg("Docker Runtime")
		manager.enableGVisor = true
	} else {
		log.Warn().Str("runtime", "default").Msg("Docker Runtime")
	}

	return manager
}

func (s *ContainerManager) AddContainer(ctx context.Context, request *Request) (containerID string, complete <-chan string, err error) {
	container := NewSandboxContainer(request, s.dockerClient)

	if s.enableGVisor {
		container.SetRuntime(docker.GVisorRuntime)
	}

	s.limiter <- request.ID
	containerID, complete, err = container.Run(ctx)

	if err != nil {
		return containerID, complete, err
	}

	s.containers.Store(containerID, container)
	return containerID, complete, nil
}

func (s *ContainerManager) RemoveContainer(ctx context.Context, containerID string, kill bool) error {
	if kill {
		if container := s.getContainer(containerID); container != nil {
			if err := s.dockerClient.ContainerKill(ctx, container.ID, "SIGKILL"); err != nil {
				return errors.Wrap(err, "failed to kill the container")
			}
		}
	}

	<-s.limiter
	s.containers.Delete(containerID)
	return nil
}

func (s *ContainerManager) GetResponse(_ context.Context, containerID string) *Response {
	if container := s.getContainer(containerID); container != nil {
		return container.GetResponse()
	}

	return nil
}

func (s *ContainerManager) getContainer(id string) *Container {
	if containerRef, ok := s.containers.Load(id); ok {
		if container, castOk := containerRef.(*Container); castOk {
			return container
		}
	}
	return nil
}

func (s *ContainerManager) Stop() {
	log.Info().Msg("stopping sandbox manager")

	atomic.StoreInt32(&s.stopFlag, 1)
}

// Start will allow the sandbox container to start listening to docker event
// stream messages allowing the start of containers to be added to the processing
func (s *ContainerManager) Start(ctx context.Context) {
	msgs, errs := s.dockerClient.Events(ctx, types.EventsOptions{
		Since: time.Now().Format(time.RFC3339),
	})

	for {
		if atomic.LoadInt32(&s.stopFlag) == 1 {
			return
		}

		select {
		case err := <-errs:
			if err != nil {
				log.Err(err).Msg("error from docker client")
			}
		case msg := <-msgs:
			if container := s.getContainer(msg.ID); container != nil {
				container.AddDockerEventMessage(&msg)
			}
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}
}
