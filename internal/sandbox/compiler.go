package sandbox

type LanguageCompiler struct {
	// The language that the given compilerName is going to be using or not. This can be seen as the kind
	// of code that is going to be executed by the requesting machine. e.g Python, Node, JavaScript,
	// C++.
	language string
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
	//  The file in which the given compilerName will be writing too (standard output and error out),
	//  since this file will be read when the response returned to the user.
	OutputFile         string
	CompilerOutputFile string
	InputFile          string
}

var Compilers = map[string]LanguageCompiler{
	"python": {
		language:           "python",
		runSteps:           "pypy /input/source",
		Interpreter:        true,
		VirtualMachineName: "virtual_machine_python",
		OutputFile:         "output",
		CompilerOutputFile: "compile",
		InputFile:          "input",
	},
	"node": {
		language:           "NodeJs",
		runSteps:           "node /input/source",
		Interpreter:        true,
		VirtualMachineName: "virtual_machine_node",
		OutputFile:         "output",
		CompilerOutputFile: "compile",
		InputFile:          "input",
	},
	"ruby": {
		language:           "Ruby",
		runSteps:           "ruby /input/source",
		Interpreter:        true,
		VirtualMachineName: "virtual_machine_ruby",
		OutputFile:         "output",
		CompilerOutputFile: "compile",
		InputFile:          "input",
	},
	"rust": {
		language: "Rust",
		runSteps: "/out",
		compileSteps: []string{
			"rustc -o /out /input/source",
		},
		Interpreter:        false,
		VirtualMachineName: "virtual_machine_rust",
		OutputFile:         "output",
		CompilerOutputFile: "compile",
		InputFile:          "input",
	},
	"go": {
		language: "Go",
		runSteps: "/out",
		compileSteps: []string{
			"cp /input/source /project/main.go",
			"go build -o /out /project/main.go",
		},
		Interpreter:        false,
		VirtualMachineName: "virtual_machine_go",
		OutputFile:         "output",
		CompilerOutputFile: "compile",
		InputFile:          "input",
	},
	"haskell": {
		language: "Haskell",
		runSteps: "/out",
		compileSteps: []string{
			"ghc -x hs -o /out /input/source",
		},
		Interpreter:        false,
		VirtualMachineName: "virtual_machine_haskell",
		OutputFile:         "output",
		CompilerOutputFile: "compile",
		InputFile:          "input",
	},
}
