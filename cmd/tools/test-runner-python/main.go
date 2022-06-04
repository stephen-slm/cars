package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/docker/docker/client"
	"github.com/google/uuid"

	"compile-and-run-sandbox/internal/sandbox"
)

func main() {
	dockerClient, _ := client.NewClientWithOpts(client.FromEnv)
	manager := sandbox.NewSandboxContainerManager(dockerClient, 5)

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		manager.Start(context.Background())
	}()

	ID, complete, err := manager.AddContainer(context.Background(), sandbox.Request{
		ID:               uuid.New().String(),
		Timeout:          1,
		MemoryConstraint: 1024,
		Path:             fmt.Sprintf("./temp/%s/", uuid.New().String()),
		SourceCode:       []string{`print("hello")sdfasdf`},
		Compiler:         sandbox.Compilers["python"],
	})

	if err == nil {
		<-complete

		resp := manager.GetResponse(context.Background(), ID)

		fmt.Println("resp.CompileMs=", resp.CompileMs)
		fmt.Println("resp.RuntimeMs=", resp.RuntimeMs)
		fmt.Println("resp.TestStatus=", resp.TestStatus)
		fmt.Println("resp.Status=", resp.Status)
		fmt.Println("resp.Output=", resp.Output)

		_ = manager.RemoveContainer(context.Background(), ID, false)
	} else {
		fmt.Println(err)
	}

	manager.Finish()

	fmt.Println("finished")

	wg.Wait()
}
