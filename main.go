package main

import (
	"github.com/docker/docker/client"
)

func main() {
	dockerClient, _ := client.NewClientWithOpts(client.FromEnv)

}
