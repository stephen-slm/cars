package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/docker/docker/client"
	"github.com/google/uuid"

	"compile-and-run-sandbox/sandbox"
)

func main() {
	dockerClient, _ := client.NewClientWithOpts(client.FromEnv)
	manager := sandbox.NewSandboxContainerManager(dockerClient)

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		manager.Start(context.Background())
	}()

	ID, complete, err := manager.AddContainer(context.Background(), sandbox.Request{
		ID:         uuid.New().String(),
		Timeout:    2,
		Path:       fmt.Sprintf("./temp/%s/", uuid.New().String()),
		SourceCode: []string{"import time\n", "time.sleep(5)\n", `print("hello")`},
		Compiler:   sandbox.Compilers[0],
		Test:       nil,
	})

	if err != nil {
		log.Fatalln(err)
	}

	<-complete

	fmt.Println("finished")

	fmt.Println(manager.GetResponse(context.Background(), ID))
	manager.Finish()

	wg.Wait()
}
