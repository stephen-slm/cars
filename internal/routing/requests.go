package routing

type CompileRequest struct {
	Language   string   `json:"language"`
	SourceCode []string `json:"source_code"`

	StdinData          []string `json:"stdin_data"`
	ExpectedStdoutData []string `json:"expected_stdout_data"`
}
