package eirinistaging_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	eirinistaging "code.cloudfoundry.org/eirini-staging"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	tmpDir      string
	cloneTarget string
	gitUrl      url.URL
	fileGitUrl  url.URL

	httpServer *httptest.Server
)

var _ = Describe("GitBuildpack", func() {
	Describe("Clone", func() {

		BeforeEach(func() {
			var err error
			gitPath, err := exec.LookPath("git")
			Expect(err).NotTo(HaveOccurred())

			tmpDir, err = ioutil.TempDir("", "tmpDir")
			Expect(err).NotTo(HaveOccurred())
			buildpackDir := filepath.Join(tmpDir, "fake-buildpack")
			err = os.MkdirAll(buildpackDir, os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			submoduleDir := filepath.Join(tmpDir, "submodule")
			err = os.MkdirAll(submoduleDir, os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			Expect(os.RemoveAll(filepath.Join(buildpackDir, ".git"))).To(Succeed())
			execute(buildpackDir, gitPath, "init")
			execute(buildpackDir, gitPath, "config", "user.email", "you@example.com")
			execute(buildpackDir, gitPath, "config", "user.name", "your name")
			writeFile(filepath.Join(buildpackDir, "content"), "some content")

			Expect(os.RemoveAll(filepath.Join(submoduleDir, ".git"))).To(Succeed())
			execute(submoduleDir, gitPath, "init")
			execute(submoduleDir, gitPath, "config", "user.email", "you@example.com")
			execute(submoduleDir, gitPath, "config", "user.name", "your name")
			writeFile(filepath.Join(submoduleDir, "README"), "1st commit")
			execute(submoduleDir, gitPath, "add", ".")
			execute(submoduleDir, gitPath, "commit", "-am", "first commit")
			writeFile(filepath.Join(submoduleDir, "README"), "2nd commit")
			execute(submoduleDir, gitPath, "commit", "-am", "second commit")

			execute(buildpackDir, gitPath, "submodule", "add", "file://"+submoduleDir, "sub")
			execute(buildpackDir+"/sub", gitPath, "checkout", "HEAD^")
			execute(buildpackDir, gitPath, "add", "-A")
			execute(buildpackDir, gitPath, "commit", "-m", "fake commit")
			execute(buildpackDir, gitPath, "commit", "--allow-empty", "-m", "empty commit")
			execute(buildpackDir, gitPath, "tag", "a_lightweight_tag")
			execute(buildpackDir, gitPath, "checkout", "-b", "a_branch")
			execute(buildpackDir+"/sub", gitPath, "checkout", "master")
			execute(buildpackDir, gitPath, "add", "-A")
			execute(buildpackDir, gitPath, "commit", "-am", "update submodule")
			execute(buildpackDir, gitPath, "checkout", "master")
			execute(buildpackDir, gitPath, "update-server-info")

			cloneTarget, err = ioutil.TempDir(tmpDir, "clone")
			Expect(err).NotTo(HaveOccurred())

			httpServer = httptest.NewServer(http.FileServer(http.Dir(tmpDir)))

			gitUrl = url.URL{
				Scheme: "http",
				Host:   httpServer.Listener.Addr().String(),
				Path:   "/fake-buildpack/.git",
			}

			fileGitUrl = url.URL{
				Scheme: "file",
				Path:   tmpDir + "/fake-buildpack",
			}
		})

		AfterEach(func() {
			// os.RemoveAll(cloneTarget)
			httpServer.Close()
			// Expect(os.RemoveAll(tmpDir)).To(Succeed())
		})

		Context("With a Git transport that doesn't support `--depth`", func() {
			It("clones a URL", func() {
				err := eirinistaging.GitClone(gitUrl, cloneTarget)
				Expect(err).NotTo(HaveOccurred())
				Expect(currentBranch(cloneTarget)).To(Equal("master"))
			})

			It("clones a URL with a branch", func() {
				branchUrl := gitUrl
				branchUrl.Fragment = "a_branch"
				err := eirinistaging.GitClone(branchUrl, cloneTarget)
				Expect(err).NotTo(HaveOccurred())
				Expect(currentBranch(cloneTarget)).To(Equal("a_branch"))
			})

			It("clones a URL with a lightweight tag", func() {
				branchUrl := gitUrl
				branchUrl.Fragment = "a_lightweight_tag"
				err := eirinistaging.GitClone(branchUrl, cloneTarget)
				Expect(err).NotTo(HaveOccurred())
				Expect(currentBranch(cloneTarget)).To(Equal("a_lightweight_tag"))
			})

			Context("when git repo has submodules", func() {
				It("updates the submodules for the branch", func() {
					branchUrl := gitUrl
					branchUrl.Fragment = "a_branch"
					err := eirinistaging.GitClone(branchUrl, cloneTarget)
					Expect(err).NotTo(HaveOccurred())

					fileContents, _ := ioutil.ReadFile(cloneTarget + "/sub/README")
					Expect(string(fileContents)).To(Equal("2nd commit"))
				})
			})

			Context("with bogus git URLs", func() {
				It("returns an error", func() {
					By("passing an invalid path", func() {
						badUrl := gitUrl
						badUrl.Path = "/a/bad/path"
						err := eirinistaging.GitClone(badUrl, cloneTarget)
						Expect(err).To(HaveOccurred())
					})

					By("passing a bad tag/branch", func() {
						badUrl := gitUrl
						badUrl.Fragment = "notfound"
						err := eirinistaging.GitClone(badUrl, cloneTarget)
						Expect(err).To(HaveOccurred())
					})
				})
			})
		})

		Context("With a Git transport that supports `--depth`", func() {
			It("clones a URL", func() {
				err := eirinistaging.GitClone(fileGitUrl, cloneTarget)
				Expect(err).NotTo(HaveOccurred())
				Expect(currentBranch(cloneTarget)).To(Equal("master"))
			})

			It("clones a URL with a branch", func() {
				branchUrl := fileGitUrl
				branchUrl.Fragment = "a_branch"
				err := eirinistaging.GitClone(branchUrl, cloneTarget)
				Expect(err).NotTo(HaveOccurred())
				Expect(currentBranch(cloneTarget)).To(Equal("a_branch"))
			})

			It("clones a URL with a lightweight tag", func() {
				branchUrl := fileGitUrl
				branchUrl.Fragment = "a_lightweight_tag"
				err := eirinistaging.GitClone(branchUrl, cloneTarget)
				Expect(err).NotTo(HaveOccurred())
				Expect(currentBranch(cloneTarget)).To(Equal("a_lightweight_tag"))
			})

			It("does a shallow clone of the repo", func() {
				gitPath, err := exec.LookPath("git")
				Expect(err).NotTo(HaveOccurred())
				version, err := exec.Command(gitPath, "version").CombinedOutput()
				Expect(err).NotTo(HaveOccurred())
				if string(version) == "git version 2.9.0\n" {
					Skip("shallow clone not support with submodules for git 2.9.0")
				}

				eirinistaging.GitClone(fileGitUrl, cloneTarget)

				cmd := exec.Command("git", "rev-list", "HEAD", "--count")
				cmd.Dir = cloneTarget
				bytes, err := cmd.Output()
				output := strings.TrimSpace(string(bytes))

				Expect(err).NotTo(HaveOccurred())
				Expect(output).To(Equal("1"))
			})
		})
	})
})

func currentBranch(gitDir string) string {
	cmd := exec.Command("git", "symbolic-ref", "--short", "-q", "HEAD")
	cmd.Dir = gitDir
	bytes, err := cmd.Output()
	if err != nil {
		// try the tag
		cmd := exec.Command("git", "name-rev", "--name-only", "--tags", "HEAD")
		cmd.Dir = gitDir
		bytes, err = cmd.Output()
	}
	Expect(err).NotTo(HaveOccurred())
	return strings.TrimSpace(string(bytes))
}

func execute(dir string, execCmd string, args ...string) {
	cmd := exec.Command(execCmd, args...)
	cmd.Dir = dir
	err := cmd.Run()
	Expect(err).NotTo(HaveOccurred())
}

func writeFile(filepath, content string) {
	err := ioutil.WriteFile(filepath,
		[]byte(content), os.ModePerm)
	Expect(err).NotTo(HaveOccurred())
}
