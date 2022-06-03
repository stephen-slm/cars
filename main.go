package main

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/docker/docker/client"
	"github.com/google/uuid"

	"compile-and-run-sandbox/sandbox"
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

	containerWg := sync.WaitGroup{}

	for i := 0; i < 10; i++ {
		containerWg.Add(1)

		go func() {
			defer containerWg.Done()

			source := `
using System;
using System.Collections.Generic;

namespace katis
{
    internal class Program
    {
        private static void Main(string[] args)
        {
            List<string> words = new List<string>();
            List<string> compoundWords = new List<string>();

            string input = Console.ReadLine();

            while (!string.IsNullOrEmpty(input))
            {
                words.AddRange(input.Split(' '));

                input = Console.ReadLine();
            }

            for (int i = 0; i < words.Count; i++)
            {
                string first = words[i];

                for (int j = 0; j < words.Count; j++)
                {
                    string second = words[j];
                    if (second == first) continue;

                    string compounded = first + second;
                    if (!compoundWords.Contains(compounded)) compoundWords.Add(compounded);
                }
            }

            compoundWords.Sort();

            foreach (string word in compoundWords)
            {
                Console.WriteLine(word);
            }

            Console.ReadLine();
        }

    }
}
			`

			ID, complete, err := manager.AddContainer(context.Background(), sandbox.Request{
				ID:               uuid.New().String(),
				Timeout:          1,
				MemoryConstraint: 1024,
				Path:             fmt.Sprintf("./temp/%s/", uuid.New().String()),
				SourceCode:       strings.Split(source, "\r\n"),
				Compiler:         sandbox.Compilers["cs"],
				Test: &sandbox.Test{
					ID:                 uuid.New().String(),
					StdinData:          []string{"a bb", "ab b"},
					ExpectedStdoutData: []string{"aab", "ab", "aba", "abb", "abbb", "ba", "bab", "bba", "bbab", "bbb"},
				},
			})

			if err == nil {
				<-complete

				resp := manager.GetResponse(context.Background(), ID)
				fmt.Println(resp)
				_ = manager.RemoveContainer(context.Background(), ID, false)
			} else {
				fmt.Println(err)
			}
		}()
	}

	containerWg.Wait()
	manager.Finish()

	fmt.Println("finished")

	wg.Wait()
}
