package sandbox

type LanguageCompiler struct {
	// The language that the given compiler is going to be using or not. This can be seen as the kind
	// of code that is going to be executed by the requesting machine. e.g Python, Node, JavaScript,
	// C++.
	language string
	// The name of the compiler that will be used to run the code. This is the name of the file that
	// will be called from the root of the docker container. e.g. node, py, python3
	compiler string
	// If the given compiler is an interpreter or not, since based on this action we would need to
	// create additional steps for compiling to a file if not.
	interpreter bool
	// The additional arguments that might be required for performing compiling actions.
	// For example letting a compiler know that they need to build first.
	AdditionalArguments string
	// This is the name of docker image that will be executed for the given code sample, this will
	// be the container that will be used for just this language. Most likely virtual_machine_language,
	// e.g. virtual_machine_python.
	VirtualMachineName string
	//  The file in which the given compiler will be writing too (standard output), since this file
	// will be read when the response returned to the user.
	StandardOutputFile string
	//  The file in which the given compiler will be writing too (error output), since this file will
	// be read when the response returned to the user.
	StandardErrorFile string
}

var Compilers = []LanguageCompiler{{
	language:            "python",
	compiler:            "python3",
	interpreter:         true,
	AdditionalArguments: "",
	VirtualMachineName:  "virtual_machine_python",
	StandardOutputFile:  "python.out",
	StandardErrorFile:   "python.error.out",
}, {
	language:            "NodeJs",
	compiler:            "node",
	interpreter:         true,
	AdditionalArguments: "",
	VirtualMachineName:  "node_virtual_machine",
	StandardOutputFile:  "node.out",
	StandardErrorFile:   "node.error.out",
}}
