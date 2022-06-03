package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/docker/docker/client"
	"github.com/google/uuid"

	"compile-and-run-sandbox/sandbox"
)

func main() {
	dockerClient, _ := client.NewClientWithOpts(client.FromEnv)
	manager := sandbox.NewSandboxContainerManager(dockerClient, 25)

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		manager.Start(context.Background())
	}()

	containerWg := sync.WaitGroup{}

	for i := 0; i < 100; i++ {
		containerWg.Add(1)

		go func() {
			defer containerWg.Done()

			ID, complete, err := manager.AddContainer(context.Background(), sandbox.Request{
				ID:         uuid.New().String(),
				Timeout:    2,
				Path:       fmt.Sprintf("./temp/%s/", uuid.New().String()),
				SourceCode: []string{`print("hello")`},
				Compiler:   sandbox.Compilers[0],
				Test: &sandbox.Test{
					ID:                 "",
					StdinData:          []string{},
					ExpectedStdoutData: []string{"hello"},
				},
			})

			if err == nil {
				<-complete

				fmt.Println(manager.GetResponse(context.Background(), ID))
				_ = manager.RemoveContainer(context.Background(), ID, false)
			}
		}()

	}

	containerWg.Wait()
	manager.Finish()

	fmt.Println("finished")

	wg.Wait()
}
