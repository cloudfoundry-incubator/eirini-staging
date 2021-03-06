package recipe_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var (
	sourcePath string
	binaries   *BinaryPaths
)

func TestRecipe(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Recipe Suite")
}

type BinaryPaths struct {
	DownloaderPath string `json:"downloader_path"`
	ExecutorPath   string `json:"executor_path"`
	UploaderPath   string `json:"uploader_path"`
}

var _ = SynchronizedBeforeSuite(func() []byte {

	sourcePath = "code.cloudfoundry.org/eirini-staging"

	downloaderPath, err := gexec.Build(filepath.Join(sourcePath, "cmd/downloader"))
	Expect(err).NotTo(HaveOccurred())

	executorPath, err := gexec.Build(filepath.Join(sourcePath, "cmd/executor"))
	Expect(err).NotTo(HaveOccurred())

	uploaderPath, err := gexec.Build(filepath.Join(sourcePath, "cmd/uploader"))
	Expect(err).NotTo(HaveOccurred())

	b := BinaryPaths{
		DownloaderPath: downloaderPath,
		ExecutorPath:   executorPath,
		UploaderPath:   uploaderPath,
	}

	bytes, err := json.Marshal(b)
	Expect(err).NotTo(HaveOccurred())

	return bytes

}, func(bytes []byte) {
	err := json.Unmarshal(bytes, &binaries)
	Expect(err).NotTo(HaveOccurred())
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	err := os.RemoveAll(binaries.DownloaderPath)
	Expect(err).NotTo(HaveOccurred())
	err = os.RemoveAll(binaries.ExecutorPath)
	Expect(err).NotTo(HaveOccurred())
	err = os.RemoveAll(binaries.UploaderPath)
	Expect(err).NotTo(HaveOccurred())
})
