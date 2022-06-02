package main

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func main() {
	dockerClient, _ := client.NewClientWithOpts(client.FromEnv)
	msgs, errs := dockerClient.Events(context.Background(), types.EventsOptions{})

	for {
		select {
		case err := <-errs:
			fmt.Println(err)
		case msg := <-msgs:
			fmt.Println(msg.ID, msg.Status)
		}
	}
}
