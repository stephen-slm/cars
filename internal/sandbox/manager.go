package sandbox

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type SandboxContainerManager struct {
	// the limiter will be a buffered channel used to determine the number of possible
	// containers that can be executed at any one time. When the container is removed
	// the buffered channel will be popped, pushing will result in a block until the
	// pop has been executed.
	limiter      chan string
	dockerClient *client.Client
	containers   sync.Map
	finished     bool
}

func NewSandboxContainerManager(dockerClient *client.Client, maxConcurrentContainers int) *SandboxContainerManager {
	return &SandboxContainerManager{
		limiter:      make(chan string, maxConcurrentContainers),
		dockerClient: dockerClient,
		containers:   sync.Map{},
	}
}

func (s *SandboxContainerManager) AddContainer(ctx context.Context, request Request) (ID string, complete <-chan string, err error) {
	container := NewSandboxContainer(request, s.dockerClient)

	s.limiter <- request.ID
	containerID, complete, err := container.Run(ctx)

	if err != nil {
		return containerID, complete, err
	}

	s.containers.Store(containerID, container)
	return containerID, complete, nil
}

func (s *SandboxContainerManager) RemoveContainer(ctx context.Context, containerID string, kill bool) error {
	if kill {
		if container, ok := s.containers.Load(containerID); ok {
			return s.dockerClient.ContainerKill(ctx, container.(*SandboxContainer).ID, "SIGKILL")
		}
	}

	<-s.limiter
	s.containers.Delete(containerID)
	return nil
}

func (s *SandboxContainerManager) GetResponse(_ context.Context, containerID string) *Response {
	if container, ok := s.containers.Load(containerID); ok {
		return container.(*SandboxContainer).GetResponse()
	}

	return nil
}

func (s *SandboxContainerManager) Finish() {
	s.finished = true
}

// Start will allow the sandbox container to start listening to docker event
// stream messages allowing the start of containers to be added to the processing
func (s *SandboxContainerManager) Start(ctx context.Context) {
	msgs, errs := s.dockerClient.Events(ctx, types.EventsOptions{
		Since: time.Now().Format(time.RFC3339),
	})

	for {
		if s.finished {
			break
		}

		select {
		case err := <-errs:
			fmt.Println(err)
		case msg := <-msgs:
			if container, ok := s.containers.Load(msg.ID); ok {
				container.(*SandboxContainer).AddDockerEventMessage(msg)
			}
		default:
		}
	}
}
