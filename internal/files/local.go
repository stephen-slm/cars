package files

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type LocalFiles struct {
	config *LocalConfig
}

type File struct {
	ID   string
	Name string
	Data []byte
}

// newLocalFiles is the local handler used during development to
// write the source code and output files to disk instead of a S3 bucket
// or another location.
func newLocalFiles(config *LocalConfig) LocalFiles {
	return LocalFiles{config: config}
}

func (l LocalFiles) WriteFile(file *File) error {
	log.Debug().
		Str("id", file.ID).
		Str("name", file.Name).
		Str("data", string(file.Data)).
		Msg("writing file locally")

	folderDirectory := filepath.Join(l.config.LocalRootPath, file.ID)
	filePath := filepath.Join(folderDirectory, file.Name)

	if err := os.MkdirAll(folderDirectory, 0o750); err != nil {
		return errors.Wrap(err, "failed to make required directories")
	}

	writeFile, writeFileErr := os.Create(filePath)

	if writeFileErr != nil {
		return errors.Wrapf(writeFileErr, "failed to create %s file", file.Name)
	}

	defer writeFile.Close()

	if _, writeErr := writeFile.Write(file.Data); writeErr != nil {
		return errors.Wrapf(writeErr, "failed to write %s", file.Name)
	}

	return nil
}

func (l LocalFiles) WriteFiles(files ...*File) []error {
	wg := sync.WaitGroup{}

	errs := make([]error, 0)
	queue := make(chan error, len(files))

	for _, file := range files {
		wg.Add(1)

		go func(file *File) {
			defer wg.Done()

			if err := l.WriteFile(file); err != nil {
				queue <- err
			}
		}(file)
	}

	wg.Wait()
	close(queue)

	for err := range queue {
		errs = append(errs, err)
	}

	return errs
}

func (l LocalFiles) GetFile(id string, name string) ([]byte, error) {
	folderDirectory := filepath.Join(l.config.LocalRootPath, id)
	filePath := filepath.Join(folderDirectory, name)

	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		return nil, errors.Wrapf(err, "cannot locate file %s", name)
	}

	data, err := os.ReadFile(filePath)

	if err != nil {
		return nil, errors.Wrapf(err, "failed to get the local file %s by id %s", name, id)
	}

	return data, nil
}
