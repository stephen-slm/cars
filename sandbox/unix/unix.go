package unix

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ConvertPathToUnix  takes a valid windows complete path and converts it to a supporting unix
// version which allows for the support of absolute paths within docker.
func ConvertPathToUnix(path string) string {
	// Convert the working directory to a unix based format. Instead of C:\\a\\b => //c//a//b//
	abs, _ := filepath.Abs(path)
	split := strings.Split(abs, ":")

	directory := split[1:]
	rootDrive := strings.ToLower(split[0])

	return strings.ReplaceAll(fmt.Sprintf("/%s%s", rootDrive, strings.Join(directory, "")), "\\", "/")
}
