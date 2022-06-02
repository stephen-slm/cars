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
}

func NewSandboxContainerManager(dockerClient *client.Client) *SandboxContainerManager {
	return &SandboxContainerManager{
		dockerClient: dockerClient,
		containers:   map[string]*SandboxContainer{},
	}
}

func (s *SandboxContainerManager) AddContainer(ctx context.Context, request Request) (string, error) {
	container := NewSandboxContainer(request, s.dockerClient)

	containerID, err := container.Run(ctx)

	if err != nil {
		return containerID, err

	}

	s.containers[containerID] = container
	return containerID, nil
}

func (s *SandboxContainerManager) RemoveContainer(ctx context.Context, containerID string, kill bool) error {
	if kill {
		if container, ok := s.containers[containerID]; ok {
			container.Kill(ctx)
		}
	}

	delete(s.containers, containerID)
}

// Start will allow the sandbox container to start listening to docker event
// stream messages allowing the start of containers to be added to the processing
func (s *SandboxContainerManager) Start(ctx context.Context) {
	msgs, errs := s.dockerClient.Events(ctx, types.EventsOptions{
		Since: time.Now().Format(time.RFC3339),
	})

	for {
		select {
		case err := <-errs:
			fmt.Println(err)
		case msg := <-msgs:
			if container, ok := s.containers[msg.ID]; ok {
				container.AddDockerEventMessage(msg)
			}
		}
	}
}
