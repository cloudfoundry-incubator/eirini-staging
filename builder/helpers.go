package builder

import (
	"crypto/md5"
	"fmt"
	"log"
	"path/filepath"
	"strings"
)

func hasFinalize(buildpackPath string) (bool, error) {
	return fileExists(filepath.Join(buildpackPath, "bin", "finalize"))
}

func hasSupply(buildpackPath string) (bool, error) {
	return fileExists(filepath.Join(buildpackPath, "bin", "supply"))
}

func BuildpackPath(baseDir, buildpackName string) string {
	return filepath.Join(baseDir, fmt.Sprintf("%x", md5.Sum([]byte(buildpackName))))
}

func IsZipFile(filename string) bool {
	return strings.HasSuffix(filename, ".zip")
}

func logError(message string) {
	log.Println(message)
}
