package files

type LocalConfig struct {
	// LocalRootPath if set will be used as the directory the source and the output
	// will be written into if its in local mode. If this is not set and local mode
	// is enabled then this will be the temp directory of the machine (os.TempDir())
	LocalRootPath string
}

type S3Config struct {
	// BucketName  is the location in which the files will be written or pulled
	// from when attempting to gather the source or the output.
	BucketName string
}

type FilesConfig struct {
	// The configuration for the local files which is used in local mode. This will only
	// be used if S3Config is not defined or local mode is enforced.
	Local *LocalConfig

	// The configuration for the S3 bucket. This will only be used if FilesConfig
	// is not defined or local mode is enforced.
	S3 *S3Config

	// If local mode should be forced or not regardless if the S3Config is configured.
	ForceLocalMode bool
}

type Files interface {
	WriteFile(id string, name string, data []byte) error
	GetFile(id string, name string) ([]byte, error)
}

func NewFilesHandler(config *FilesConfig) (Files, error) {
	if config.ForceLocalMode || config.S3 == nil || config.S3.BucketName == "" {
		return newLocalFiles(config.Local)
	}

	return newS3Files(config.S3)
}
