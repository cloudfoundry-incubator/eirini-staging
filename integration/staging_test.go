package recipe_test

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"strconv"

	archive_helpers "code.cloudfoundry.org/archiver/extractor/test_helper"
	eirinistaging "code.cloudfoundry.org/eirini-staging"
	"code.cloudfoundry.org/eirini-staging/builder"
	"code.cloudfoundry.org/eirini-staging/integration/integrationfakes"
	"code.cloudfoundry.org/urljoiner"

	"code.cloudfoundry.org/bbs/models"
	"code.cloudfoundry.org/tlsconfig"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

//go:generate counterfeiter net/http.HandlerFunc

var _ = Describe("Staging Test", func() {

	const (
		stagingGUID        = "staging-guid"
		completionCallback = ""
		responseURL        = "/stage/staging-guid/completed"
	)

	var (
		err            error
		server         *ghttp.Server
		eiriniServer   *ghttp.Server
		appbitBytes    []byte
		buildpackBytes []byte
		buildpacksDir  string
		workspaceDir   string
		outputDir      string
		cacheDir       string
		certsPath      string
		actualBytes    []byte
		expectedBytes  []byte
	)

	createTempDir := func() string {
		tempDir, createErr := ioutil.TempDir("", "")
		Expect(createErr).NotTo(HaveOccurred())
		Expect(chownR(tempDir, "vcap", "vcap")).To(Succeed())

		return tempDir
	}

	createTestServer := func(certName, keyName, caCertName string) *ghttp.Server {
		certPath := filepath.Join(certsPath, certName)
		keyPath := filepath.Join(certsPath, keyName)
		caCertPath := filepath.Join(certsPath, caCertName)

		tlsConf, tlsErr := tlsconfig.Build(
			tlsconfig.WithInternalServiceDefaults(),
			tlsconfig.WithIdentityFromFile(certPath, keyPath),
		).Server(
			tlsconfig.WithClientAuthenticationFromFile(caCertPath),
		)
		Expect(tlsErr).NotTo(HaveOccurred())

		testServer := ghttp.NewUnstartedServer()
		testServer.HTTPTestServer.TLS = tlsConf

		return testServer
	}

	rubyBuildpack := func() builder.Buildpack {
		return builder.Buildpack{
			Name: "ruby_buildpack",
			Key:  "ruby_buildpack",
			URL:  "https://github.com/cloudfoundry/ruby-buildpack",
		}
	}

	myBuildpack := func() builder.Buildpack {
		return builder.Buildpack{
			Name: "my_buildpack",
			Key:  "my_buildpack",
			URL:  urljoiner.Join(server.URL(), "/my-buildpack.zip"),
		}
	}

	myBuildpackWithSkipDetect := func() builder.Buildpack {
		mybp := myBuildpack()
		mybp.SkipDetect = true
		return mybp
	}

	badurlBuildpack := func() builder.Buildpack {
		return builder.Buildpack{
			Name: "badurl_buildpack",
			Key:  "bardurl_buildpack",
			URL:  "bad-url.zip",
		}
	}

	BeforeEach(func() {
		Expect(os.Setenv(eirinistaging.EnvBuildpackCacheDir, os.TempDir())).To(Succeed())
		workspaceDir = createTempDir()
		outputDir = createTempDir()
		cacheDir = createTempDir()
		buildpacksDir = createTempDir()

		Expect(os.Setenv(eirinistaging.EnvWorkspaceDir, workspaceDir)).To(Succeed())
		Expect(os.Setenv(eirinistaging.EnvOutputDropletLocation, path.Join(outputDir, "droplet.tgz"))).To(Succeed())
		Expect(os.Setenv(eirinistaging.EnvOutputMetadataLocation, path.Join(outputDir, "result.json"))).To(Succeed())
		Expect(os.Setenv("CF_STACK", "cflinuxfs3")).To(Succeed())
		Expect(os.Setenv(eirinistaging.EnvOutputBuildArtifactsCache, path.Join(cacheDir, "cache.tgz"))).To(Succeed())
		Expect(os.Setenv(eirinistaging.EnvBuildpacksDir, buildpacksDir)).To(Succeed())
		Expect(os.Setenv(eirinistaging.EnvStagingGUID, stagingGUID)).To(Succeed())
		Expect(os.Setenv(eirinistaging.EnvCompletionCallback, completionCallback)).To(Succeed())
		Expect(os.Setenv(eirinistaging.EnvBuildArtifactsCacheDir, path.Join(cacheDir, "cache"))).To(Succeed())
		Expect(os.Setenv(eirinistaging.EnvBuildpackCacheDownloadURI, "")).To(Succeed())
		Expect(os.Setenv(eirinistaging.EnvBuildpackCacheChecksum, "")).To(Succeed())
		Expect(os.Setenv(eirinistaging.EnvBuildpackCacheChecksumAlgorithm, "")).To(Succeed())
		Expect(os.Setenv(eirinistaging.EnvBuildpackCacheUploadURI, "")).To(Succeed())

		certsPath, err = filepath.Abs("testdata/certs")
		Expect(err).NotTo(HaveOccurred())

		server = createTestServer("cc-server-crt", "cc-server-crt-key", "internal-ca-cert")
		eiriniServer = createTestServer("eirini.crt", "eirini.key", "internal-ca-cert")
	})

	AfterEach(func() {
		Expect(os.RemoveAll(buildpacksDir)).To(Succeed())
		Expect(os.RemoveAll(workspaceDir)).To(Succeed())
		Expect(os.RemoveAll(outputDir)).To(Succeed())
		Expect(os.RemoveAll(cacheDir)).To(Succeed())

		Expect(os.Unsetenv("CF_STACK")).To(Succeed())
		Expect(os.Unsetenv(eirinistaging.EnvCertsPath)).To(Succeed())
		Expect(os.Unsetenv(eirinistaging.EnvStagingGUID)).To(Succeed())
		Expect(os.Unsetenv(eirinistaging.EnvCompletionCallback)).To(Succeed())
		Expect(os.Unsetenv(eirinistaging.EnvBuildpacksDir)).To(Succeed())
		Expect(os.Unsetenv(eirinistaging.EnvBuildpacks)).To(Succeed())
		Expect(os.Unsetenv(eirinistaging.EnvDownloadURL)).To(Succeed())
		Expect(os.Unsetenv(eirinistaging.EnvWorkspaceDir)).To(Succeed())
		Expect(os.Unsetenv(eirinistaging.EnvOutputDropletLocation)).To(Succeed())
		Expect(os.Unsetenv(eirinistaging.EnvOutputMetadataLocation)).To(Succeed())
		Expect(os.Unsetenv(eirinistaging.EnvOutputBuildArtifactsCache)).To(Succeed())
		Expect(os.Unsetenv(eirinistaging.EnvEiriniAddress)).To(Succeed())
		Expect(os.Unsetenv(eirinistaging.EnvBuildArtifactsCacheDir)).To(Succeed())
		Expect(os.Unsetenv(eirinistaging.EnvBuildpackCacheDownloadURI)).To(Succeed())
		Expect(os.Unsetenv(eirinistaging.EnvBuildpackCacheChecksum)).To(Succeed())
		Expect(os.Unsetenv(eirinistaging.EnvBuildpackCacheChecksumAlgorithm)).To(Succeed())
		Expect(os.Unsetenv("TMPDIR")).To(Succeed())

		server.Close()
		eiriniServer.Close()
	})

	Describe("download", func() {
		var downloaderSession *gexec.Session

		Describe("buildpack as git repo", func() {
			BeforeEach(func() {
				appbitBytes, err = ioutil.ReadFile("testdata/dora.zip")
				Expect(err).NotTo(HaveOccurred())

				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/my-app-bits"),
						ghttp.RespondWith(http.StatusOK, appbitBytes),
					),
				)
				server.HTTPTestServer.StartTLS()
				eiriniServer.HTTPTestServer.StartTLS()

				Expect(os.Setenv(eirinistaging.EnvEiriniAddress, eiriniServer.URL())).To(Succeed())
				Expect(os.Setenv(eirinistaging.EnvDownloadURL, urljoiner.Join(server.URL(), "my-app-bits"))).To(Succeed())
				Expect(os.Setenv(eirinistaging.EnvBuildpacks, buildpacksJSON(rubyBuildpack()))).To(Succeed())
				Expect(os.Setenv(eirinistaging.EnvCertsPath, certsPath)).To(Succeed())
			})

			JustBeforeEach(func() {
				downloaderSession = runDownloader()
			})

			It("runs successfully", func() {
				Expect(downloaderSession.ExitCode()).To(BeZero())
			})

			Context("prints the staging log", func() {
				It("should print the installation log", func() {
					Expect(downloaderSession.Err).To(gbytes.Say("Installing dependencies"))
				})
			})
		})

		Describe("buildpack as zip archive", func() {
			BeforeEach(func() {
				appbitBytes, err = ioutil.ReadFile("testdata/dora.zip")
				Expect(err).NotTo(HaveOccurred())

				buildpackBytes, err = ioutil.ReadFile("testdata/ruby-buildpack-cflinuxfs3-v1.8.0.zip")
				Expect(err).NotTo(HaveOccurred())

				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/my-buildpack.zip"),
						ghttp.RespondWith(http.StatusOK, buildpackBytes),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/my-app-bits"),
						ghttp.RespondWith(http.StatusOK, appbitBytes),
					),
				)
				server.HTTPTestServer.StartTLS()
				eiriniServer.HTTPTestServer.StartTLS()

				Expect(os.Setenv(eirinistaging.EnvEiriniAddress, eiriniServer.URL())).To(Succeed())
				Expect(os.Setenv(eirinistaging.EnvDownloadURL, urljoiner.Join(server.URL(), "my-app-bits"))).To(Succeed())
				Expect(os.Setenv(eirinistaging.EnvBuildpacks, buildpacksJSON(myBuildpack()))).To(Succeed())
				Expect(os.Setenv(eirinistaging.EnvCertsPath, certsPath)).To(Succeed())
			})

			JustBeforeEach(func() {
				downloaderSession = runDownloader()
			})

			It("runs successfully", func() {
				Expect(downloaderSession.ExitCode()).To(BeZero())
			})

			It("installs the buildpack json", func() {
				expectedFile := filepath.Join(buildpacksDir, "config.json")
				Expect(expectedFile).To(BeARegularFile())
			})

			It("installs the buildpack", func() {
				md5Hash := fmt.Sprintf("%x", md5.Sum([]byte("my_buildpack")))
				expectedBuildpackPath := path.Join(buildpacksDir, md5Hash)
				Expect(expectedBuildpackPath).To(BeADirectory())
			})

			It("places the app bits in the workspace", func() {
				actualBytes, err = ioutil.ReadFile(path.Join(workspaceDir, eirinistaging.AppBits))
				Expect(err).NotTo(HaveOccurred())
				expectedBytes, err = ioutil.ReadFile("testdata/dora.zip")
				Expect(err).NotTo(HaveOccurred())
				Expect(actualBytes).To(Equal(expectedBytes))
			})

			Context("fails", func() {
				BeforeEach(func() {
					Expect(os.Setenv(eirinistaging.EnvBuildpacks, buildpacksJSON(badurlBuildpack()))).To(Succeed())

					eiriniServer.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("PUT", responseURL),
							verifyResponse(true, "failed to request buildpack"),
						),
					)
				})

				It("should send completion response with a failure", func() {
					Expect(eiriniServer.ReceivedRequests()).To(HaveLen(1))
				})

				It("should exit with non-zero exit code", func() {
					Expect(downloaderSession.ExitCode()).NotTo(BeZero())
				})
			})
		})

		Describe("buildpack cache", func() {
			var buildpackCacheBytes []byte
			BeforeEach(func() {
				appbitBytes, err = ioutil.ReadFile("testdata/dora.zip")
				Expect(err).NotTo(HaveOccurred())
				buildpackCacheBytes, err = ioutil.ReadFile("testdata/buildpack-cache.tgz")
				Expect(err).NotTo(HaveOccurred())

				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/my-app-bits"),
						ghttp.RespondWith(http.StatusOK, appbitBytes),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/my-buildpack-cache"),
						ghttp.RespondWith(http.StatusOK, buildpackCacheBytes),
					),
				)
				server.HTTPTestServer.StartTLS()
				eiriniServer.HTTPTestServer.StartTLS()

				Expect(os.Setenv(eirinistaging.EnvEiriniAddress, eiriniServer.URL())).To(Succeed())
				Expect(os.Setenv(eirinistaging.EnvDownloadURL, urljoiner.Join(server.URL(), "my-app-bits"))).To(Succeed())
				Expect(os.Setenv(eirinistaging.EnvBuildpackCacheDownloadURI, urljoiner.Join(server.URL(), "my-buildpack-cache"))).To(Succeed())
				Expect(os.Setenv(eirinistaging.EnvBuildpacks, buildpacksJSON(rubyBuildpack()))).To(Succeed())
				Expect(os.Setenv(eirinistaging.EnvCertsPath, certsPath)).To(Succeed())

				buildpackCacheChecksum := sha256ForBytes(buildpackCacheBytes)
				Expect(os.Setenv(eirinistaging.EnvBuildpackCacheChecksum, buildpackCacheChecksum)).To(Succeed())
				Expect(os.Setenv(eirinistaging.EnvBuildpackCacheChecksumAlgorithm, "sha256")).To(Succeed())
			})

			JustBeforeEach(func() {
				downloaderSession = runDownloader()
			})

			It("runs successfully", func() {
				Expect(downloaderSession.ExitCode()).To(BeZero())
			})

			It("downloads the buildpack cache", func() {
				downloadedCacheFilePath := path.Join(os.Getenv(eirinistaging.EnvBuildArtifactsCacheDir), "app.zip")
				Expect(downloadedCacheFilePath).To(BeAnExistingFile())

				donwloadedCacheBytes, readErr := ioutil.ReadFile(downloadedCacheFilePath)
				Expect(readErr).NotTo(HaveOccurred())
				Expect(donwloadedCacheBytes).To(Equal(buildpackCacheBytes))
			})

			It("unpacks the cache", func() {
				buildpackCacheFilePath := path.Join(os.Getenv(eirinistaging.EnvBuildArtifactsCacheDir), "buildpack-cache", "cache")
				Expect(buildpackCacheFilePath).To(BeAnExistingFile())
			})

			Context("when the buildpack cache checksum cannot be verified", func() {
				BeforeEach(func() {
					Expect(os.Setenv(eirinistaging.EnvBuildpackCacheChecksum, "trololo")).To(Succeed())
					eiriniServer.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("PUT", responseURL),
							verifyResponse(true, "checksum verification failure"),
						),
					)
				})

				It("should exit with non-zero exit code", func() {
					Expect(downloaderSession.ExitCode()).NotTo(BeZero())
				})
			})

			Context("when the buildpack cache checksum algorithm is not supported", func() {
				BeforeEach(func() {
					Expect(os.Setenv(eirinistaging.EnvBuildpackCacheChecksumAlgorithm, "lil-sha")).To(Succeed())
					eiriniServer.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("PUT", responseURL),
							verifyResponse(true, "algorithm"),
						),
					)
				})

				It("should exit with non-zero exit code", func() {
					Expect(downloaderSession.ExitCode()).NotTo(BeZero())
				})
			})
		})
	})

	Describe("execute", func() {
		var executorSession *gexec.Session

		Context("when extract fails", func() {
			BeforeEach(func() {
				appbitBytes, err = ioutil.ReadFile("testdata/bad-dora.zip")
				Expect(err).NotTo(HaveOccurred())

				buildpackBytes, err = ioutil.ReadFile("testdata/ruby-buildpack-cflinuxfs3-v1.8.0.zip")
				Expect(err).NotTo(HaveOccurred())
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/my-buildpack.zip"),
						ghttp.RespondWith(http.StatusOK, buildpackBytes),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/my-app-bits"),
						ghttp.RespondWith(http.StatusOK, appbitBytes),
					),
				)
				eiriniServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("PUT", responseURL),
						verifyResponse(true, "not a valid zip file"),
					),
				)
				server.HTTPTestServer.StartTLS()
				eiriniServer.HTTPTestServer.StartTLS()

				Expect(os.Setenv(eirinistaging.EnvEiriniAddress, eiriniServer.URL())).To(Succeed())
				Expect(os.Setenv(eirinistaging.EnvDownloadURL, urljoiner.Join(server.URL(), "my-app-bits"))).To(Succeed())
				Expect(os.Setenv(eirinistaging.EnvBuildpacks, buildpacksJSON(myBuildpack()))).To(Succeed())
				Expect(os.Setenv(eirinistaging.EnvCertsPath, certsPath)).To(Succeed())

				runDownloader()
			})

			JustBeforeEach(func() {
				executorSession = runExecutor()
			})

			It("should send completion response with a failure", func() {
				Expect(executorSession.ExitCode()).NotTo(BeZero())
				Expect(eiriniServer.ReceivedRequests()).To(HaveLen(1))
			})
		})

		Describe("buildpack is a git repo", func() {
			BeforeEach(func() {
				appbitBytes, err = ioutil.ReadFile("testdata/dora.zip")
				Expect(err).NotTo(HaveOccurred())

				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/my-app-bits"),
						ghttp.RespondWith(http.StatusOK, appbitBytes),
					),
				)
				server.HTTPTestServer.StartTLS()
				eiriniServer.HTTPTestServer.StartTLS()

				Expect(os.Setenv(eirinistaging.EnvEiriniAddress, eiriniServer.URL())).To(Succeed())
				Expect(os.Setenv(eirinistaging.EnvDownloadURL, urljoiner.Join(server.URL(), "my-app-bits"))).To(Succeed())
				Expect(os.Setenv(eirinistaging.EnvBuildpacks, buildpacksJSON(rubyBuildpack()))).To(Succeed())
				Expect(os.Setenv(eirinistaging.EnvCertsPath, certsPath)).To(Succeed())

				runDownloader()
			})

			JustBeforeEach(func() {
				executorSession = runExecutor()
			})

			It("should create the droplet and output metadata", func() {
				Expect(executorSession.ExitCode()).To(BeZero())

				Expect(path.Join(outputDir, "droplet.tgz")).To(BeARegularFile())
				Expect(path.Join(outputDir, "result.json")).To(BeARegularFile())
			})
		})

		Describe("extract succeeds", func() {
			BeforeEach(func() {
				appbitBytes, err = ioutil.ReadFile("testdata/dora.zip")
				Expect(err).NotTo(HaveOccurred())

				buildpackBytes, err = ioutil.ReadFile("testdata/ruby-buildpack-cflinuxfs3-v1.8.0.zip")
				Expect(err).NotTo(HaveOccurred())
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/my-buildpack.zip"),
						ghttp.RespondWith(http.StatusOK, buildpackBytes),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/my-app-bits"),
						ghttp.RespondWith(http.StatusOK, appbitBytes),
					),
				)
				server.HTTPTestServer.StartTLS()
				eiriniServer.HTTPTestServer.StartTLS()

				Expect(os.Setenv(eirinistaging.EnvEiriniAddress, eiriniServer.URL())).To(Succeed())
				Expect(os.Setenv(eirinistaging.EnvDownloadURL, urljoiner.Join(server.URL(), "my-app-bits"))).To(Succeed())
				Expect(os.Setenv(eirinistaging.EnvBuildpacks, buildpacksJSON(myBuildpack()))).To(Succeed())
				Expect(os.Setenv(eirinistaging.EnvCertsPath, certsPath)).To(Succeed())

				runDownloader()
			})

			JustBeforeEach(func() {
				executorSession = runExecutor()
			})

			It("should create the droplet and output metadata", func() {
				Expect(executorSession.ExitCode()).To(BeZero())

				Expect(path.Join(outputDir, "droplet.tgz")).To(BeARegularFile())
				Expect(path.Join(outputDir, "result.json")).To(BeARegularFile())
			})

			Context("fails", func() {
				BeforeEach(func() {
					Expect(os.Setenv(eirinistaging.EnvWorkspaceDir, filepath.Join(workspaceDir, "bad-workspace-dir"))).To(Succeed())
					eiriniServer.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("PUT", responseURL),
							verifyResponse(true, "failed to create droplet"),
						),
					)
				})

				It("should send completion response with a failure", func() {
					Expect(eiriniServer.ReceivedRequests()).To(HaveLen(1))
				})

				It("should exit with non-zero exit code", func() {
					Expect(executorSession.ExitCode()).NotTo(BeZero())
				})
			})
		})

		Describe("the loggregator app", func() {
			BeforeEach(func() {
				appbitBytes, err = ioutil.ReadFile("testdata/logapp.zip")
				Expect(err).NotTo(HaveOccurred())

				buildpackBytes, err = ioutil.ReadFile("testdata/ruby-buildpack-cflinuxfs3-v1.8.0.zip")
				Expect(err).NotTo(HaveOccurred())
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/my-buildpack.zip"),
						ghttp.RespondWith(http.StatusOK, buildpackBytes),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/my-app-bits"),
						ghttp.RespondWith(http.StatusOK, appbitBytes),
					),
				)
				server.HTTPTestServer.StartTLS()
				eiriniServer.HTTPTestServer.StartTLS()

				Expect(os.Setenv(eirinistaging.EnvEiriniAddress, eiriniServer.URL())).To(Succeed())
				Expect(os.Setenv(eirinistaging.EnvDownloadURL, urljoiner.Join(server.URL(), "my-app-bits"))).To(Succeed())
				Expect(os.Setenv(eirinistaging.EnvBuildpacks, buildpacksJSON(myBuildpackWithSkipDetect()))).To(Succeed())
				Expect(os.Setenv(eirinistaging.EnvCertsPath, certsPath)).To(Succeed())

				runDownloader()
			})

			JustBeforeEach(func() {
				executorSession = runExecutor()
			})

			It("should create the droplet and output metadata", func() {
				Expect(executorSession.ExitCode()).To(BeZero())

				Expect(path.Join(outputDir, "droplet.tgz")).To(BeARegularFile())
				Expect(path.Join(outputDir, "result.json")).To(BeARegularFile())
			})
		})

		Describe("execute fails", func() {
			var (
				buildpackPath string
				tmpdir        string
			)

			JustBeforeEach(func() {
				appbitBytes, err = ioutil.ReadFile("testdata/catnip.zip")
				Expect(err).NotTo(HaveOccurred())

				buildpackBytes, err = ioutil.ReadFile(buildpackPath)
				Expect(err).NotTo(HaveOccurred())
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/my-buildpack.zip"),
						ghttp.RespondWith(http.StatusOK, buildpackBytes),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/my-app-bits"),
						ghttp.RespondWith(http.StatusOK, appbitBytes),
					),
				)
				server.HTTPTestServer.StartTLS()
				eiriniServer.HTTPTestServer.StartTLS()

				Expect(os.Setenv(eirinistaging.EnvEiriniAddress, eiriniServer.URL())).To(Succeed())
				Expect(os.Setenv(eirinistaging.EnvDownloadURL, urljoiner.Join(server.URL(), "my-app-bits"))).To(Succeed())
			})

			AfterEach(func() {
				Expect(os.RemoveAll(tmpdir)).To(Succeed())
			})

			Context("when detect fails", func() {
				BeforeEach(func() {
					tmpdir, err = ioutil.TempDir(os.TempDir(), "matching-buildpack")
					Expect(err).ToNot(HaveOccurred())

					buildpackPath = path.Join(tmpdir, "buildpack.zip")
					archive_helpers.CreateZipArchive(buildpackPath, []archive_helpers.ArchiveFile{
						{
							Name: "bin/detect",
							Body: `#!/bin/bash

  exit 1
`,
						},
					})
				})

				JustBeforeEach(func() {
					Expect(os.Setenv(eirinistaging.EnvBuildpacks, buildpacksJSON(myBuildpack()))).To(Succeed())
					Expect(os.Setenv(eirinistaging.EnvCertsPath, certsPath)).To(Succeed())

					runDownloader()
				})

				It("should fail with exit code 222", func() {
					eiriniServer.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("PUT", responseURL),
							verifyResponse(true, "NoAppDetectedError: exit status 222"),
						),
					)

					Expect(runExecutor().ExitCode()).To(Equal(builder.DetectFailCode))
				})
			})

			Context("when compile fails", func() {
				BeforeEach(func() {
					tmpdir, err = ioutil.TempDir(os.TempDir(), "matching-buildpack")
					Expect(err).ToNot(HaveOccurred())

					buildpackPath = path.Join(tmpdir, "buildpack.zip")
					archive_helpers.CreateZipArchive(buildpackPath, []archive_helpers.ArchiveFile{
						{
							Name: "bin/compile",
							Body: `#!/bin/bash

  exit 1
`,
						},
					})
				})

				JustBeforeEach(func() {
					Expect(os.Setenv(eirinistaging.EnvBuildpacks, buildpacksJSON(myBuildpackWithSkipDetect()))).To(Succeed())

					Expect(os.Setenv(eirinistaging.EnvCertsPath, certsPath)).To(Succeed())

					runDownloader()
				})

				It("should fail with exit code 223", func() {
					eiriniServer.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("PUT", responseURL),
							verifyResponse(true, "BuildpackCompileFailed: exit status 223"),
						),
					)

					Expect(runExecutor().ExitCode()).To(Equal(builder.CompileFailCode))
				})
			})

			Context("when release fails", func() {
				BeforeEach(func() {
					tmpdir, err = ioutil.TempDir(os.TempDir(), "matching-buildpack")
					Expect(err).ToNot(HaveOccurred())

					buildpackPath = path.Join(tmpdir, "buildpack.zip")
					archive_helpers.CreateZipArchive(buildpackPath, []archive_helpers.ArchiveFile{
						{
							Name: "bin/compile",
							Body: `#!/bin/bash

  exit 0
`,
						},
						{
							Name: "bin/release",
							Body: `#!/bin/bash

  exit 1
`,
						},
					})
				})

				JustBeforeEach(func() {
					Expect(os.Setenv(eirinistaging.EnvBuildpacks, buildpacksJSON(myBuildpackWithSkipDetect()))).To(Succeed())
					Expect(os.Setenv(eirinistaging.EnvCertsPath, certsPath)).To(Succeed())

					runDownloader()
				})

				It("should fail with exit code 224", func() {
					eiriniServer.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("PUT", responseURL),
							verifyResponse(true, "BuildpackReleaseFailed: exit status 224"),
						),
					)

					Expect(runExecutor().ExitCode()).To(Equal(builder.ReleaseFailCode))
				})
			})

		})
	})

	Describe("binary buildpack", func() {
		var executorSession *gexec.Session

		Context("when extract succeeds", func() {
			BeforeEach(func() {
				appbitBytes, err = ioutil.ReadFile("testdata/catnip.zip")
				Expect(err).NotTo(HaveOccurred())

				buildpackBytes, err = ioutil.ReadFile("testdata/binary-buildpack-cflinuxfs3-v1.0.34.zip")
				Expect(err).NotTo(HaveOccurred())
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/my-buildpack.zip"),
						ghttp.RespondWith(http.StatusOK, buildpackBytes),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/my-app-bits"),
						ghttp.RespondWith(http.StatusOK, appbitBytes),
					),
				)
				server.HTTPTestServer.StartTLS()
				eiriniServer.HTTPTestServer.StartTLS()

				Expect(os.Setenv(eirinistaging.EnvEiriniAddress, eiriniServer.URL())).To(Succeed())
				Expect(os.Setenv(eirinistaging.EnvDownloadURL, urljoiner.Join(server.URL(), "my-app-bits"))).To(Succeed())
				Expect(os.Setenv(eirinistaging.EnvBuildpacks, buildpacksJSON(myBuildpackWithSkipDetect()))).To(Succeed())
				Expect(os.Setenv(eirinistaging.EnvCertsPath, certsPath)).To(Succeed())

				runDownloader()
			})

			JustBeforeEach(func() {
				executorSession = runExecutor()
			})

			It("should create the droplet and output metadata", func() {
				Expect(executorSession.ExitCode()).To(BeZero())

				Expect(path.Join(outputDir, "droplet.tgz")).To(BeARegularFile())
				Expect(path.Join(outputDir, "result.json")).To(BeARegularFile())
			})
		})
	})

	Describe("upload", func() {
		var uploaderSession *gexec.Session

		BeforeEach(func() {
			appbitBytes, err = ioutil.ReadFile("testdata/dora.zip")
			Expect(err).NotTo(HaveOccurred())

			buildpackBytes, err = ioutil.ReadFile("testdata/ruby-buildpack-cflinuxfs3-v1.8.0.zip")
			Expect(err).NotTo(HaveOccurred())

			server.AppendHandlers(
				// Downloader
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/my-buildpack.zip"),
					ghttp.RespondWith(http.StatusOK, buildpackBytes),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/my-app-bits"),
					ghttp.RespondWith(http.StatusOK, appbitBytes),
				),

				// Uploader
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/my-droplet"),
					ghttp.RespondWith(http.StatusOK, ""),
				),
			)

			eiriniServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", responseURL),
					ghttp.RespondWith(http.StatusOK, ""),
					ghttp.VerifyMimeType("application/json"),
					verifyResponse(false, "ruby"),
				),
			)

			server.HTTPTestServer.StartTLS()
			eiriniServer.HTTPTestServer.StartTLS()

			Expect(os.Setenv(eirinistaging.EnvEiriniAddress, eiriniServer.URL())).To(Succeed())
			Expect(os.Setenv(eirinistaging.EnvDownloadURL, urljoiner.Join(server.URL(), "my-app-bits"))).To(Succeed())
			Expect(os.Setenv(eirinistaging.EnvDropletUploadURL, urljoiner.Join(server.URL(), "my-droplet"))).To(Succeed())
			Expect(os.Setenv(eirinistaging.EnvBuildpacks, buildpacksJSON(myBuildpack()))).To(Succeed())
			Expect(os.Setenv(eirinistaging.EnvCertsPath, certsPath)).To(Succeed())

			runDownloader()

			runExecutor()
		})

		JustBeforeEach(func() {
			uploaderSession = runUploader()
		})

		It("should successfully upload the droplet", func() {
			Expect(uploaderSession).To(gexec.Exit(0))
		})

		Context("fails", func() {
			BeforeEach(func() {
				Expect(os.Setenv(eirinistaging.EnvOutputDropletLocation, path.Join(outputDir, "bad-location.tgz"))).To(Succeed())

				eiriniServer.SetHandler(0,
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("PUT", responseURL),
						verifyResponse(true, "no such file"),
					),
				)
			})

			It("should send completion response with a failure", func() {
				Expect(eiriniServer.ReceivedRequests()).To(HaveLen(1))
			})

			It("should return an error", func() {
				Expect(uploaderSession.ExitCode()).NotTo(BeZero())
			})
		})

		Context("and eirini returns response with failure status", func() {
			BeforeEach(func() {
				eiriniServer.SetHandler(0,
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("PUT", responseURL),
						ghttp.RespondWith(http.StatusInternalServerError, ""),
					),
				)
			})

			It("should return an error", func() {
				Expect(server.ReceivedRequests()).To(HaveLen(3))
				Expect(eiriniServer.ReceivedRequests()).To(HaveLen(1))

				Expect(uploaderSession.ExitCode()).NotTo(BeZero())
			})
		})

		Context("when the buildpack cache upload uri is set", func() {
			var (
				cacheFilePath           string
				verifyInvokationHandler *integrationfakes.FakeHandlerFunc
			)

			BeforeEach(func() {
				verifyInvokationHandler = new(integrationfakes.FakeHandlerFunc)
				cacheFilePath = path.Join(cacheDir, "buildpack-cache-to-upload")
				Expect(ioutil.WriteFile(cacheFilePath, []byte("random-buildpack-stuff"), 0755)).To(Succeed())
				Expect(os.Setenv(eirinistaging.EnvOutputBuildArtifactsCache, cacheFilePath)).To(Succeed())
				Expect(os.Setenv(eirinistaging.EnvBuildpackCacheUploadURI, urljoiner.Join(server.URL(), "bpcache-upload"))).To(Succeed())

				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/bpcache-upload"),
						ghttp.RespondWith(http.StatusOK, ""),
						ghttp.VerifyBody([]byte("random-buildpack-stuff")),
						ghttp.CombineHandlers(verifyInvokationHandler.Spy),
					),
				)
			})

			It("succeeds", func() {
				Expect(uploaderSession).To(gexec.Exit(0))
			})

			It("uploads the buildpack cache", func() {
				Expect(verifyInvokationHandler.CallCount()).To(Equal(1))
			})
		})
	})
})

func verifyResponse(failed bool, reason string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		body, err := ioutil.ReadAll(req.Body)
		req.Body.Close()
		Expect(err).ShouldNot(HaveOccurred())

		var uploaderResponse models.TaskCallbackResponse
		Expect(json.Unmarshal(body, &uploaderResponse)).To(Succeed())
		Expect(uploaderResponse.Failed).To(Equal(failed))
		if failed {
			Expect(uploaderResponse.FailureReason).To(ContainSubstring(reason))
		} else {
			Expect(uploaderResponse.Result).To(ContainSubstring(reason))
		}
	}
}

func chownR(path, username, group string) error {
	uid, gid, err := getIds(username, group)
	if err != nil {
		return err
	}

	return filepath.Walk(path, func(name string, info os.FileInfo, err error) error {
		if err == nil {
			err = os.Chown(name, uid, gid)
		}
		return err
	})
}

func getIds(username, group string) (uid int, gid int, err error) {
	var g *user.Group
	g, err = user.LookupGroup(group)
	if err != nil {
		return -1, -1, err
	}

	var u *user.User
	u, err = user.Lookup(username)
	if err != nil {
		return -1, -1, err
	}

	uid, err = strconv.Atoi(u.Uid)
	if err != nil {
		return -1, -1, err
	}

	gid, err = strconv.Atoi(g.Gid)
	if err != nil {
		return -1, -1, err
	}

	return uid, gid, nil
}

func sha256ForBytes(b []byte) string {
	checksum := sha256.New()
	_, err := io.Copy(checksum, bytes.NewReader(b))
	Expect(err).NotTo(HaveOccurred())
	return fmt.Sprintf("%x", checksum.Sum(nil))
}

func buildpacksJSON(buildpacks ...builder.Buildpack) string {
	bytes, err := json.Marshal(buildpacks)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return string(bytes)
}

func runDownloader() *gexec.Session {
	cmd := exec.Command(binaries.DownloaderPath)
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	EventuallyWithOffset(1, session, 60).Should(gexec.Exit())
	return session
}

func runExecutor() *gexec.Session {
	cmd := exec.Command(binaries.ExecutorPath)
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	EventuallyWithOffset(1, session, 600).Should(gexec.Exit())
	return session
}

func runUploader() *gexec.Session {
	cmd := exec.Command(binaries.UploaderPath)
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	Eventually(session, 10).Should(gexec.Exit())
	return session
}
