package eirinistaging_test

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	eirinistaging "code.cloudfoundry.org/eirini-staging"
	"code.cloudfoundry.org/eirini-staging/builder"
)

var _ = Describe("Buildpackmanager", func() {

	var (
		client           *http.Client
		buildpackDir     string
		buildpacksJSON   []byte
		buildpackManager eirinistaging.Installer
		buildpacks       []builder.Buildpack
		server           *ghttp.Server
		responseContent  []byte
		err              error
	)

	BeforeEach(func() {
		client = http.DefaultClient

		buildpackDir, err = ioutil.TempDir("", "buildpacks")
		Expect(err).ToNot(HaveOccurred())

		responseContent, err = makeZippedPackage()
		Expect(err).ToNot(HaveOccurred())

		server = ghttp.NewServer()
		server.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/my-buildpack"),
				ghttp.RespondWith(http.StatusOK, responseContent),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/your-buildpack"),
				ghttp.RespondWith(http.StatusOK, responseContent),
			),
		)

	})

	JustBeforeEach(func() {
		buildpacksJSON, err = json.Marshal(buildpacks)
		Expect(err).NotTo(HaveOccurred())

		buildpackManager = eirinistaging.NewBuildpackManager(client, client, buildpackDir, string(buildpacksJSON))
		err = buildpackManager.Install()
	})

	Context("When a list of Buildpacks needs be installed", func() {
		BeforeEach(func() {
			buildpacks = []builder.Buildpack{
				{
					Name: "my_buildpack",
					Key:  "my-key",
					URL:  fmt.Sprintf("%s/my-buildpack", server.URL()),
				},
				{
					Name: "your_buildpack",
					Key:  "your-key",
					URL:  fmt.Sprintf("%s/your-buildpack", server.URL()),
				},
			}
		})

		It("should not fail", func() {
			Expect(err).ToNot(HaveOccurred())
		})

		It("should download all buildpacks to the given directory", func() {
			myMd5Dir := fmt.Sprintf("%x", md5.Sum([]byte("my_buildpack")))
			yourMd5Dir := fmt.Sprintf("%x", md5.Sum([]byte("your_buildpack")))
			Expect(filepath.Join(buildpackDir, myMd5Dir)).To(BeADirectory())
			Expect(filepath.Join(buildpackDir, yourMd5Dir)).To(BeADirectory())
		})

		It("should write a config.json file in the correct location", func() {
			Expect(filepath.Join(buildpackDir, "config.json")).To(BeAnExistingFile())
		})

		It("marshals the provided buildpacks to the config.json", func() {
			var actualBytes []byte
			actualBytes, err = ioutil.ReadFile(filepath.Join(buildpackDir, "config.json"))
			Expect(err).ToNot(HaveOccurred())

			var actualBuildpacks []builder.Buildpack
			err = json.Unmarshal(actualBytes, &actualBuildpacks)
			Expect(err).ToNot(HaveOccurred())
			Expect(buildpacks).To(ConsistOf(actualBuildpacks))
		})
	})

	Context("When a single buildpack with skip detect is provided", func() {
		BeforeEach(func() {
			buildpacks = []builder.Buildpack{
				{
					Name:       "my_buildpack",
					Key:        "my-key",
					URL:        fmt.Sprintf("%s/my-buildpack", server.URL()),
					SkipDetect: true,
				},
			}
		})

		It("should not fail", func() {
			Expect(err).ToNot(HaveOccurred())
		})

		It("should download all buildpacks to the given directory", func() {
			myMd5Dir := fmt.Sprintf("%x", md5.Sum([]byte("my_buildpack")))
			Expect(filepath.Join(buildpackDir, myMd5Dir)).To(BeADirectory())
		})

		It("should write a config.json file in the correct location", func() {
			Expect(filepath.Join(buildpackDir, "config.json")).To(BeAnExistingFile())
		})

		It("marshals the provided buildpacks to the config.json", func() {
			var actualBytes []byte
			actualBytes, err = ioutil.ReadFile(filepath.Join(buildpackDir, "config.json"))
			Expect(err).ToNot(HaveOccurred())

			var actualBuildpacks []builder.Buildpack
			err = json.Unmarshal(actualBytes, &actualBuildpacks)
			Expect(err).ToNot(HaveOccurred())
			Expect(buildpacks).To(ConsistOf(actualBuildpacks))
		})
	})

	Context("When the buildpack url is invalid", func() {
		BeforeEach(func() {
			server = ghttp.NewServer()
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/bad-buildpack.zip"),
					ghttp.RespondWith(http.StatusInternalServerError, responseContent),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/bad-buildpack.zip"),
					ghttp.RespondWith(http.StatusInternalServerError, responseContent),
				),
			)

			buildpacks = []builder.Buildpack{
				{
					Name: "bad_buildpack",
					Key:  "bad-key",
					URL:  fmt.Sprintf("%s/bad-buildpack.zip", server.URL()),
				},
			}
		})

		It("should try both http clients", func() {
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("default client also failed")))
		})
	})

	Context("When the buildpack file is invalid zip", func() {
		BeforeEach(func() {
			server = ghttp.NewServer()
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/bad-buildpack.zip"),
					ghttp.RespondWith(http.StatusOK, []byte{}),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/bad-buildpack.zip/info/refs"),
					ghttp.RespondWith(http.StatusNotFound, nil),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/bad-buildpack.zip/info/refs"),
					ghttp.RespondWith(http.StatusNotFound, nil),
				),
			)

			buildpacks = []builder.Buildpack{
				{
					Name: "bad_buildpack",
					Key:  "bad-key",
					URL:  fmt.Sprintf("%s/bad-buildpack.zip", server.URL()),
				},
			}
		})

		It("should try http client and also git clone", func() {
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("Failed to clone git repository")))
		})
	})

	Context("when the buildpack url is a git repo", func() {
		var (
			tmpDir      string
			cloneTarget string
			gitURL      url.URL
			gitPath     string
		)

		BeforeEach(func() {
			gitPath, err = exec.LookPath("git")
			Expect(err).NotTo(HaveOccurred())

			tmpDir, err = ioutil.TempDir("", "tmpDir")
			Expect(err).NotTo(HaveOccurred())
			gitBuildpackDir := filepath.Join(tmpDir, "fake-buildpack")
			err = os.MkdirAll(gitBuildpackDir, os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			submoduleDir := filepath.Join(tmpDir, "submodule")
			err = os.MkdirAll(submoduleDir, os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			Expect(os.RemoveAll(filepath.Join(gitBuildpackDir, ".git"))).To(Succeed())
			execute(gitBuildpackDir, gitPath, "init")
			execute(gitBuildpackDir, gitPath, "config", "user.email", "you@example.com")
			execute(gitBuildpackDir, gitPath, "config", "user.name", "your name")
			writeFile(filepath.Join(gitBuildpackDir, "content"), "some content")

			Expect(os.RemoveAll(filepath.Join(submoduleDir, ".git"))).To(Succeed())
			execute(submoduleDir, gitPath, "init")
			execute(submoduleDir, gitPath, "config", "user.email", "you@example.com")
			execute(submoduleDir, gitPath, "config", "user.name", "your name")
			writeFile(filepath.Join(submoduleDir, "README"), "1st commit")
			execute(submoduleDir, gitPath, "add", ".")
			execute(submoduleDir, gitPath, "commit", "-am", "first commit")
			writeFile(filepath.Join(submoduleDir, "README"), "2nd commit")
			execute(submoduleDir, gitPath, "commit", "-am", "second commit")

			execute(gitBuildpackDir, gitPath, "submodule", "add", "file://"+submoduleDir, "sub")
			execute(gitBuildpackDir+"/sub", gitPath, "checkout", "HEAD^")
			execute(gitBuildpackDir, gitPath, "add", "-A")
			execute(gitBuildpackDir, gitPath, "commit", "-m", "fake commit")
			execute(gitBuildpackDir, gitPath, "commit", "--allow-empty", "-m", "empty commit")
			execute(gitBuildpackDir, gitPath, "tag", "a_lightweight_tag")
			execute(gitBuildpackDir, gitPath, "checkout", "-b", "a_branch")
			execute(gitBuildpackDir+"/sub", gitPath, "checkout", "master")
			execute(gitBuildpackDir, gitPath, "add", "-A")
			execute(gitBuildpackDir, gitPath, "commit", "-am", "update submodule")
			execute(gitBuildpackDir, gitPath, "checkout", "master")
			execute(gitBuildpackDir, gitPath, "update-server-info")

			httpServer = httptest.NewServer(http.FileServer(http.Dir(tmpDir)))

		})

		AfterEach(func() {
			os.RemoveAll(cloneTarget)
			httpServer.Close()
			Expect(os.RemoveAll(tmpDir)).To(Succeed())
		})

		Context("with a valid url", func() {
			BeforeEach(func() {
				gitURL = url.URL{
					Scheme: "http",
					Host:   httpServer.Listener.Addr().String(),
					Path:   "/fake-buildpack/.git",
				}

				buildpacks = []builder.Buildpack{
					{
						Name: "buildpack",
						Key:  "key",
						URL:  gitURL.String(),
					},
				}

				cloneTarget = builder.BuildpackPath(buildpackDir, buildpacks[0].Name)
			})

			It("should succeed cloning the buildpack", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
