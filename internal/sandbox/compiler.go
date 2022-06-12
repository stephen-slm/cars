package sandbox

import (
	"embed"
	"fmt"

	"github.com/rs/zerolog/log"
)

type LanguageCompiler struct {
	// The language that the given compilerName is going to be using or not. This can be seen as the kind
	// of code that is going to be executed by the requesting machine. e.g Python, Node, JavaScript,
	// C++.
	Language string
	// The name of the compiler.
	Compiler string
	// The name of the compiler used in the steps, this is not including the actions which
	// are used to perform the compilation and running.
	runSteps string
	// The steps used to compile the application, these are skipped if interpreter is true.
	compileSteps []string
	// If the given compilerName is an interpreter or not, since based on this action we would need to
	// create additional steps for compiling to a file if not.
	Interpreter bool
	// This is the name of docker image that will be executed for the given code sample, this will
	// be the container that will be used for just this language. Most likely virtual_machine_language,
	// e.g. virtual_machine_python.
	VirtualMachineName string
	SourceFile         string
	//  The file in which the given compilerName will be writing too (standard output and error out),
	//  since this file will be read when the response returned to the user.
	OutputFile         string
	CompilerOutputFile string
	InputFile          string
}

var Compilers = map[string]*LanguageCompiler{
	"python2": {
		Language:           "Python 2 (pypy)",
		runSteps:           "pypy /input/solution.py",
		Interpreter:        true,
		VirtualMachineName: "virtual_machine_python2",
		SourceFile:         "solution.py",
		OutputFile:         "output",
		CompilerOutputFile: "compile",
		InputFile:          "input",
	},
	"python": {
		Language:           "Python (pypy)",
		runSteps:           "pypy /input/solution.py",
		Interpreter:        true,
		VirtualMachineName: "virtual_machine_python",
		SourceFile:         "solution.py",
		OutputFile:         "output",
		CompilerOutputFile: "compile",
		InputFile:          "input",
	},
	"node": {
		Language:           "NodeJs (Javascript)",
		runSteps:           "node /input/solution.js",
		Interpreter:        true,
		VirtualMachineName: "virtual_machine_node",
		SourceFile:         "solution.js",
		OutputFile:         "output",
		CompilerOutputFile: "compile",
		InputFile:          "input",
	},
	"ruby": {
		Language:           "Ruby",
		runSteps:           "ruby /input/solution.rb",
		Interpreter:        true,
		VirtualMachineName: "virtual_machine_ruby",
		SourceFile:         "solution.rb",
		OutputFile:         "output",
		CompilerOutputFile: "compile",
		InputFile:          "input",
	},
	"rust": {
		Compiler: "rustc",
		Language: "Rust",
		runSteps: "/solution",
		compileSteps: []string{
			"rustc -o /solution /input/solution.rs",
		},
		Interpreter:        false,
		VirtualMachineName: "virtual_machine_rust",
		SourceFile:         "solution.rs",
		OutputFile:         "output",
		CompilerOutputFile: "compile",
		InputFile:          "input",
	},
	"go": {
		Compiler: "go",
		Language: "Golang",
		runSteps: "/solution",
		compileSteps: []string{
			"cp /input/solution.go /project/main.go",
			"go build -o /solution /project/main.go",
		},
		Interpreter:        false,
		VirtualMachineName: "virtual_machine_go",
		SourceFile:         "solution.go",
		OutputFile:         "output",
		CompilerOutputFile: "compile",
		InputFile:          "input",
	},
	"haskell": {
		Compiler: "ghc",
		Language: "Haskell",
		runSteps: "/solution",
		compileSteps: []string{
			"ghc -o /solution /input/solution.hs",
		},
		Interpreter:        false,
		VirtualMachineName: "virtual_machine_haskell",
		SourceFile:         "solution.hs",
		OutputFile:         "output",
		CompilerOutputFile: "compile",
		InputFile:          "input",
	},
	"c": {
		Compiler: "gcc",
		Language: "C",
		runSteps: "/solution",
		compileSteps: []string{
			"gcc -g -O2 -std=gnu11 -static -o /solution /input/solution.c -lm",
		},
		Interpreter:        false,
		VirtualMachineName: "virtual_machine_gcc",
		SourceFile:         "solution.c",
		OutputFile:         "output",
		CompilerOutputFile: "compile",
		InputFile:          "input",
	},
	"cpp": {
		Compiler: "gcc",
		Language: "C++",
		runSteps: "/solution",
		compileSteps: []string{
			"g++ -g -O2 -std=gnu++17 -static -lrt -Wl,--whole-archive -lpthread -Wl,--no-whole-archive -o /solution /input/solution.cpp",
		},
		Interpreter:        false,
		VirtualMachineName: "virtual_machine_gcc",
		SourceFile:         "solution.cpp",
		OutputFile:         "output",
		CompilerOutputFile: "compile",
		InputFile:          "input",
	},
	"fsharp": {
		Compiler: "dotnet6",
		Language: "F#",
		runSteps: "/build-output/projectf",
		compileSteps: []string{
			"cp /input/solution.fs /projectf/Program.fs",
			"dotnet build --configuration Release -o /build-output/ /projectf/",
		},
		Interpreter:        false,
		VirtualMachineName: "virtual_machine_dotnet6",
		SourceFile:         "solution.fs",
		OutputFile:         "output",
		CompilerOutputFile: "compile",
		InputFile:          "input",
	},
	"csharp": {
		Compiler: "dotnet6",
		Language: "C#",
		runSteps: "/build-output/projectc",
		compileSteps: []string{
			"cp /input/solution.cs /projectc/Program.cs",
			"dotnet build --configuration Release -o /build-output/ /projectc/",
		},
		Interpreter:        false,
		VirtualMachineName: "virtual_machine_dotnet6",
		SourceFile:         "source.cs",
		OutputFile:         "output",
		CompilerOutputFile: "compile",
		InputFile:          "input",
	},
}

//go:embed templates/*
var content embed.FS

// CompilerTemplate - this will be filled with the template data for API calls.
// the data should be small so templates will be in memory always.
var CompilerTemplate = map[string]string{}

func LoadEmbededFiles() {
	for s := range Compilers {
		data, err := content.ReadFile(fmt.Sprintf("templates/%s.txt", s))

		if err != nil {
			log.Warn().Str("lang", s).Msg("language does not have a template")
			continue
		}

		CompilerTemplate[s] = string(data)
	}
}
