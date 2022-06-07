package files

type Files interface {
	WriteFile(id string, name string, data []byte) error
	GetFile(id string, name string) ([]byte, error)
}
