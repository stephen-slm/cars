package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/namsral/flag"

	"compile-and-run-sandbox/internal/sandbox"
)

var ENDCOLOR = "\033[0m"
var RED = "\033[31m"
var GREEN = "\033[32m"

func main() {
	if runtime.GOOS == "windows" {
		RED = ""
		ENDCOLOR = ""
		GREEN = ""
	}

	var (
		filterName string
		verbose    bool
	)

	flag.StringVar(&filterName, "clang", "", "")
	flag.BoolVar(&verbose, "v", false, "")

	flag.Parse()

	if strings.TrimSpace(filterName) != "" {
		c, ok := sandbox.Compilers[filterName]

		if !ok {
			log.Fatalf("language '%s' does not exist in supported compilers\n", filterName)
		}

		if c.Compiler != "" {
			runDockerCommand(filterName, c.Compiler, verbose)
		} else {
			runDockerCommand(filterName, filterName, verbose)
		}

		return
	}

	completed := map[string]bool{}

	for langRef, c := range sandbox.Compilers {
		if _, ok := completed[c.VirtualMachineName]; ok {
			continue
		}

		dockerFilePrefix := strings.Split(c.VirtualMachineName, "_")[2]
		runDockerCommand(langRef, dockerFilePrefix, verbose)
	}
}

func runDockerCommand(lang string, name string, verbose bool) {
	fmt.Printf("%sRunning language:%s %s%s%s\n", RED, ENDCOLOR, GREEN, lang, ENDCOLOR)

	path := fmt.Sprintf("./build/dockerfiles/%s.dockerfile", name)
	machineName := fmt.Sprintf("virtual_machine_%s", name)

	cmd := exec.Command("docker", "build", "-f", path, "-t", machineName)

	cmd.Stdout = nil
	cmd.Stderr = os.Stderr

	if verbose {
		cmd.Args = append(cmd.Args, "--progress=plain")
		cmd.Stdout = os.Stdout
	}

	cmd.Args = append(cmd.Args, ".")

	err := cmd.Run()

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%sFinished language:%s %s%s%s\n", RED, ENDCOLOR, GREEN, lang, ENDCOLOR)
}
