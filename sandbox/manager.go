package sandbox

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type SandboxContainerManager struct {
	dockerClient *client.Client
	containers   map[string]*SandboxContainer
	finished     bool
}

func NewSandboxContainerManager(dockerClient *client.Client) *SandboxContainerManager {
	return &SandboxContainerManager{
		dockerClient: dockerClient,
		containers:   map[string]*SandboxContainer{},
	}
}

func (s *SandboxContainerManager) AddContainer(ctx context.Context, request Request) (ID string, complete <-chan string, err error) {
	container := NewSandboxContainer(request, s.dockerClient)

	containerID, complete, err := container.Run(ctx)

	if err != nil {
		return containerID, complete, err

	}

	s.containers[containerID] = container
	return containerID, complete, nil
}

func (s *SandboxContainerManager) RemoveContainer(ctx context.Context, containerID string, kill bool) error {
	if kill {
		if container, ok := s.containers[containerID]; ok {
			return s.dockerClient.ContainerKill(ctx, container.ID, "SIGKILL")
		}
	}

	delete(s.containers, containerID)
	return nil
}

func (s *SandboxContainerManager) GetResponse(ctx context.Context, containerID string) *Response {
	if container, ok := s.containers[containerID]; ok {
		return container.GetResponse()
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
			fmt.Println(msg.ID, msg.Status)

			if container, ok := s.containers[msg.ID]; ok {
				container.AddDockerEventMessage(msg)
			}
		}
	}
}
