package builder

import (
	"crypto/md5" // #nosec G501
	"fmt"
	"log"
	"path/filepath"
)

func hasFinalize(buildpackPath string) (bool, error) {
	return fileExists(filepath.Join(buildpackPath, "bin", "finalize"))
}

func hasSupply(buildpackPath string) (bool, error) {
	return fileExists(filepath.Join(buildpackPath, "bin", "supply"))
}

func BuildpackPath(baseDir, buildpackName string) string {
	return filepath.Join(baseDir, fmt.Sprintf("%x", md5.Sum([]byte(buildpackName)))) // #nosec G401
}

func logError(message string) {
	log.Println(message)
}
