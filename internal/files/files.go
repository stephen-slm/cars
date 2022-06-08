package files

type Files interface {
	WriteFile(id string, name string, data []byte) error
	GetFile(id string, name string) ([]byte, error)
}

type FilesConfig struct {
	// LocalRootPath if set will be used as the directory the source and the output
	// will be written into if its in local mode. If this is not set and local mode
	// is enabled then this will be the temp directory of the machine (os.TempDir())
	LocalRootPath string
}
