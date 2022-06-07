package files

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

type LocalFiles struct {
	rootPath string
}

// NewLocalFileHandler is the local handler used during development to
// write the source code and output files to disk instead of a S3 bucket
// or another location.
func NewLocalFileHandler(rootPath string) LocalFiles {
	return LocalFiles{rootPath: rootPath}
}

func (l LocalFiles) WriteFile(id string, name string, data []byte) error {
	folderDirectory := filepath.Join(l.rootPath, id)
	filePath := filepath.Join(folderDirectory, name)

	if err := os.MkdirAll(folderDirectory, 0o750); err != nil {
		return errors.Wrap(err, "failed to make required directories")
	}

	writeFile, writeFileErr := os.Create(filePath)

	if writeFileErr != nil {
		return errors.Wrapf(writeFileErr, "failed to create %s file", name)
	}

	defer writeFile.Close()

	if _, writeErr := writeFile.Write(data); writeErr != nil {
		return errors.Wrapf(writeErr, "failed to write %s", name)
	}

	return nil
}

func (l LocalFiles) GetFile(id string, name string) ([]byte, error) {
	folderDirectory := filepath.Join(l.rootPath, id)
	filePath := filepath.Join(folderDirectory, name)

	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		return nil, errors.Wrapf(err, "cannot locate file %s", name)
	}

	return os.ReadFile(filePath)
}
