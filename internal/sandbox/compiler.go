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
	interpreter bool
	// This is the name of docker image that will be executed for the given code sample, this will
	// be the container that will be used for just this language. Most likely virtual_machine_language,
	// e.g. virtual_machine_python.
	VirtualMachineName string
	//  The file in which the given compilerName will be writing too (standard output and error out),
	//  since this file will be read when the response returned to the user.
	OutputFile string
	InputFile  string
}

var Compilers = map[string]LanguageCompiler{
	"python": {
		language:           "python",
		runSteps:           "python /input/source",
		interpreter:        true,
		VirtualMachineName: "virtual_machine_python",
		OutputFile:         "output",
		InputFile:          "input",
	},
	"node": {
		language:           "NodeJs",
		runSteps:           "node /input/source",
		interpreter:        true,
		VirtualMachineName: "virtual_machine_node",
		OutputFile:         "output",
		InputFile:          "input",
	},
	"cs": {
		language: "cs",
		compileSteps: []string{
			"mv /input/source /project/Program.cs",
			"dotnet build -c Release --no-restore -o /out /project",
		},
		runSteps:           "dotnet /out/project.dll",
		interpreter:        false,
		VirtualMachineName: "virtual_machine_cs",
		OutputFile:         "output",
		InputFile:          "input",
	},
}
