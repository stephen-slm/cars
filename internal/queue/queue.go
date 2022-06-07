package queue

type CompileMessage struct {
	ID                 string   `json:"id"`
	Language           string   `json:"language" validate:"required,oneof=python node"`
	StdinData          []string `json:"stdin_data" validate:"required"`
	ExpectedStdoutData []string `json:"expected_stdout_data" validate:"required"`
}
