package recipe_test

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
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
	"code.cloudfoundry.org/urljoiner"

	"code.cloudfoundry.org/bbs/models"
	"code.cloudfoundry.org/tlsconfig"
	"github.com/onsi/gomega/gexec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("StagingText", func() {

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
		session        *gexec.Session
		buildpacks     []builder.Buildpack
		buildpacksDir  string
		workspaceDir   string
		outputDir      string
		cacheDir       string
		certsPath      string
		buildpackJSON  []byte
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

		tlsConf, err := tlsconfig.Build(
			tlsconfig.WithInternalServiceDefaults(),
			tlsconfig.WithIdentityFromFile(certPath, keyPath),
		).Server(
			tlsconfig.WithClientAuthenticationFromFile(caCertPath),
		)
		Expect(err).NotTo(HaveOccurred())

		testServer := ghttp.NewUnstartedServer()
		testServer.HTTPTestServer.TLS = tlsConf

		return testServer
	}

	BeforeEach(func() {
		workspaceDir = createTempDir()
		outputDir = createTempDir()
		cacheDir = createTempDir()
		buildpacksDir = createTempDir()

		Expect(os.Setenv(eirinistaging.EnvWorkspaceDir, workspaceDir)).To(Succeed())
		Expect(os.Setenv(eirinistaging.EnvOutputDropletLocation, path.Join(outputDir, "droplet.tgz"))).To(Succeed())
		Expect(os.Setenv(eirinistaging.EnvOutputMetadataLocation, path.Join(outputDir, "result.json"))).To(Succeed())
		Expect(os.Setenv("CF_STACK", "cflinuxfs2")).To(Succeed())
		Expect(os.Setenv(eirinistaging.EnvOutputBuildArtifactsCache, path.Join(cacheDir, "cache.tgz"))).To(Succeed())
		Expect(os.Setenv(eirinistaging.EnvBuildpacksDir, buildpacksDir)).To(Succeed())
		Expect(os.Setenv(eirinistaging.EnvStagingGUID, stagingGUID)).To(Succeed())
		Expect(os.Setenv(eirinistaging.EnvCompletionCallback, completionCallback)).To(Succeed())

		certsPath, err = filepath.Abs("testdata/certs")
		Expect(err).NotTo(HaveOccurred())

		server = createTestServer("cc-server-crt", "cc-server-crt-key", "internal-ca-cert")
		eiriniServer = createTestServer("eirini.crt", "eirini.key", "clientCA.crt")
	})

	AfterEach(func() {
		Expect(os.RemoveAll(buildpacksDir)).To(Succeed())
		Expect(os.RemoveAll(workspaceDir)).To(Succeed())
		Expect(os.RemoveAll(outputDir)).To(Succeed())
		Expect(os.RemoveAll(cacheDir)).To(Succeed())

		Expect(os.Unsetenv("cflinuxfs2")).To(Succeed())
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
		server.Close()
		eiriniServer.Close()
	})

	Context("when a droplet needs building...", func() {
		Context("download", func() {
			Context("with buildpack as git repo", func() {
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

					buildpacks = []builder.Buildpack{
						{
							Name: "app_buildpack",
							Key:  "app_buildpack",
							URL:  "https://github.com/cloudfoundry/ruby-buildpack",
						},
					}

					buildpackJSON, err = json.Marshal(buildpacks)
					Expect(err).ToNot(HaveOccurred())

					Expect(os.Setenv(eirinistaging.EnvBuildpacks, string(buildpackJSON))).To(Succeed())

					Expect(os.Setenv(eirinistaging.EnvCertsPath, certsPath)).To(Succeed())
				})

				JustBeforeEach(func() {
					cmd := exec.Command(binaries.DownloaderPath)
					session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
					Eventually(session, 60).Should(gexec.Exit())
				})

				It("runs successfully", func() {
					Expect(session.ExitCode()).To(BeZero())
				})
			})

			Context("with buildpack as zip archive", func() {
				BeforeEach(func() {
					appbitBytes, err = ioutil.ReadFile("testdata/dora.zip")
					Expect(err).NotTo(HaveOccurred())

					buildpackBytes, err = ioutil.ReadFile("testdata/ruby-buildpack-cflinuxfs2-v1.7.35.zip")
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

					buildpacks = []builder.Buildpack{
						{
							Name: "app_buildpack",
							Key:  "app_buildpack",
							URL:  urljoiner.Join(server.URL(), "/my-buildpack.zip"),
						},
					}

					buildpackJSON, err = json.Marshal(buildpacks)
					Expect(err).ToNot(HaveOccurred())

					Expect(os.Setenv(eirinistaging.EnvBuildpacks, string(buildpackJSON))).To(Succeed())

					Expect(os.Setenv(eirinistaging.EnvCertsPath, certsPath)).To(Succeed())
				})

				JustBeforeEach(func() {
					cmd := exec.Command(binaries.DownloaderPath)
					session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
					Eventually(session).Should(gexec.Exit())
				})

				It("runs successfully", func() {
					Expect(session.ExitCode()).To(BeZero())
				})

				It("installs the buildpack json", func() {
					expectedFile := filepath.Join(buildpacksDir, "config.json")
					Expect(expectedFile).To(BeARegularFile())
				})

				It("installs the buildpack", func() {
					md5Hash := fmt.Sprintf("%x", md5.Sum([]byte("app_buildpack")))
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
						buildpacks = []builder.Buildpack{
							{
								Name: "app_buildpack",
								Key:  "app_buildpack",
								URL:  "bad-url.zip",
							},
						}

						buildpackJSON, err = json.Marshal(buildpacks)
						Expect(err).ToNot(HaveOccurred())

						Expect(os.Setenv(eirinistaging.EnvBuildpacks, string(buildpackJSON))).To(Succeed())

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
						Expect(session.ExitCode).NotTo(BeZero())
					})
				})
			})
		})

		Context("execute", func() {
			Context("when extract fails", func() {
				BeforeEach(func() {
					appbitBytes, err = ioutil.ReadFile("testdata/bad-dora.zip")
					Expect(err).NotTo(HaveOccurred())

					buildpackBytes, err = ioutil.ReadFile("testdata/ruby-buildpack-cflinuxfs2-v1.7.35.zip")
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

					buildpacks = []builder.Buildpack{
						{
							Name: "ruby_buildpack",
							Key:  "ruby_buildpack",
							URL:  urljoiner.Join(server.URL(), "/my-buildpack.zip"),
						},
					}

					buildpackJSON, err = json.Marshal(buildpacks)
					Expect(err).ToNot(HaveOccurred())

					Expect(os.Setenv(eirinistaging.EnvBuildpacks, string(buildpackJSON))).To(Succeed())

					Expect(os.Setenv(eirinistaging.EnvCertsPath, certsPath)).To(Succeed())

					cmd := exec.Command(binaries.DownloaderPath)
					session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
					Expect(err).ToNot(HaveOccurred())
					Eventually(session).Should(gexec.Exit())
				})

				JustBeforeEach(func() {
					cmd := exec.Command(binaries.ExecutorPath)
					session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
					Eventually(session, 80).Should(gexec.Exit())
				})

				It("should send completion response with a failure", func() {
					Expect(session.ExitCode).NotTo(BeZero())
					Expect(eiriniServer.ReceivedRequests()).To(HaveLen(1))
				})
			})

			Context("when buildpack is a git repo", func() {
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

					buildpacks = []builder.Buildpack{
						{
							Name: "app_buildpack",
							Key:  "app_buildpack",
							URL:  "https://github.com/cloudfoundry/ruby-buildpack",
						},
					}

					buildpackJSON, err = json.Marshal(buildpacks)
					Expect(err).ToNot(HaveOccurred())

					Expect(os.Setenv(eirinistaging.EnvBuildpacks, string(buildpackJSON))).To(Succeed())

					Expect(os.Setenv(eirinistaging.EnvCertsPath, certsPath)).To(Succeed())

					cmd := exec.Command(binaries.DownloaderPath)
					session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
					Eventually(session, 30).Should(gexec.Exit())
					Expect(err).NotTo(HaveOccurred())

				})

				JustBeforeEach(func() {
					cmd := exec.Command(binaries.ExecutorPath)
					session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
					Eventually(session, 600).Should(gexec.Exit())
				})

				It("should create the droplet and output metadata", func() {
					Expect(session.ExitCode()).To(BeZero())

					Expect(path.Join(outputDir, "droplet.tgz")).To(BeARegularFile())
					Expect(path.Join(outputDir, "result.json")).To(BeARegularFile())
				})
			})

			Context("when extract succeeds", func() {
				BeforeEach(func() {
					appbitBytes, err = ioutil.ReadFile("testdata/dora.zip")
					Expect(err).NotTo(HaveOccurred())

					buildpackBytes, err = ioutil.ReadFile("testdata/ruby-buildpack-cflinuxfs2-v1.7.35.zip")
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

					buildpacks = []builder.Buildpack{
						{
							Name: "ruby_buildpack",
							Key:  "ruby_buildpack",
							URL:  urljoiner.Join(server.URL(), "/my-buildpack.zip"),
						},
					}

					buildpackJSON, err = json.Marshal(buildpacks)
					Expect(err).ToNot(HaveOccurred())

					Expect(os.Setenv(eirinistaging.EnvBuildpacks, string(buildpackJSON))).To(Succeed())

					Expect(os.Setenv(eirinistaging.EnvCertsPath, certsPath)).To(Succeed())

					cmd := exec.Command(binaries.DownloaderPath)
					session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
					Expect(err).ToNot(HaveOccurred())
					Eventually(session).Should(gexec.Exit())
				})

				JustBeforeEach(func() {
					cmd := exec.Command(binaries.ExecutorPath)
					session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
					Eventually(session, 80).Should(gexec.Exit())
				})

				It("should create the droplet and output metadata", func() {
					Expect(session.ExitCode()).To(BeZero())

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
						Expect(session.ExitCode).NotTo(BeZero())
					})
				})
			})

			Context("with the loggregator app", func() {
				BeforeEach(func() {
					appbitBytes, err = ioutil.ReadFile("testdata/logapp.zip")
					Expect(err).NotTo(HaveOccurred())

					buildpackBytes, err = ioutil.ReadFile("testdata/ruby-buildpack-cflinuxfs2-v1.7.35.zip")
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

					buildpacks = []builder.Buildpack{
						{
							Name:       "ruby_buildpack",
							Key:        "ruby_buildpack",
							URL:        urljoiner.Join(server.URL(), "/my-buildpack.zip"),
							SkipDetect: true,
						},
					}

					buildpackJSON, err = json.Marshal(buildpacks)
					Expect(err).ToNot(HaveOccurred())

					Expect(os.Setenv(eirinistaging.EnvBuildpacks, string(buildpackJSON))).To(Succeed())

					Expect(os.Setenv(eirinistaging.EnvCertsPath, certsPath)).To(Succeed())

					cmd := exec.Command(binaries.DownloaderPath)
					session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
					Expect(err).ToNot(HaveOccurred())
					Eventually(session).Should(gexec.Exit())
				})

				JustBeforeEach(func() {
					cmd := exec.Command(binaries.ExecutorPath)
					session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
					Eventually(session, 80).Should(gexec.Exit())
				})

				It("should create the droplet and output metadata", func() {
					Expect(session.ExitCode()).To(BeZero())

					Expect(path.Join(outputDir, "droplet.tgz")).To(BeARegularFile())
					Expect(path.Join(outputDir, "result.json")).To(BeARegularFile())
				})
			})

			Context("when execute fails", func() {
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
								Body: fmt.Sprintf(`#!/bin/bash

  exit 1
`),
							},
						})
					})

					JustBeforeEach(func() {
						buildpacks = []builder.Buildpack{
							{
								Name: "ruby_buildpack",
								Key:  "ruby_buildpack",
								URL:  urljoiner.Join(server.URL(), "/my-buildpack.zip"),
							},
						}

						buildpackJSON, err = json.Marshal(buildpacks)
						Expect(err).ToNot(HaveOccurred())

						Expect(os.Setenv(eirinistaging.EnvBuildpacks, string(buildpackJSON))).To(Succeed())

						Expect(os.Setenv(eirinistaging.EnvCertsPath, certsPath)).To(Succeed())

						cmd := exec.Command(binaries.DownloaderPath)
						session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						Eventually(session).Should(gexec.Exit())
					})

					It("should fail with exit code 222", func() {
						eiriniServer.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.VerifyRequest("PUT", responseURL),
								verifyResponse(true, "NoAppDetectedError: exit status 222"),
							),
						)

						cmd := exec.Command(binaries.ExecutorPath)
						session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
						Eventually(session, 80).Should(gexec.Exit())
						Expect(session.ExitCode()).To(Equal(builder.DetectFailCode))
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
								Body: fmt.Sprintf(`#!/bin/bash

  exit 1
`),
							},
						})
					})

					JustBeforeEach(func() {
						buildpacks = []builder.Buildpack{
							{
								Name:       "ruby_buildpack",
								Key:        "ruby_buildpack",
								URL:        urljoiner.Join(server.URL(), "/my-buildpack.zip"),
								SkipDetect: true,
							},
						}

						buildpackJSON, err = json.Marshal(buildpacks)
						Expect(err).ToNot(HaveOccurred())

						Expect(os.Setenv(eirinistaging.EnvBuildpacks, string(buildpackJSON))).To(Succeed())

						Expect(os.Setenv(eirinistaging.EnvCertsPath, certsPath)).To(Succeed())

						cmd := exec.Command(binaries.DownloaderPath)
						session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						Eventually(session).Should(gexec.Exit())
					})

					It("should fail with exit code 223", func() {
						eiriniServer.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.VerifyRequest("PUT", responseURL),
								verifyResponse(true, "BuildpackCompileFailed: exit status 223"),
							),
						)

						cmd := exec.Command(binaries.ExecutorPath)
						session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
						Eventually(session, 80).Should(gexec.Exit())
						Expect(session.ExitCode()).To(Equal(builder.CompileFailCode))
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
								Body: fmt.Sprintf(`#!/bin/bash

  exit 0
`),
							},
							{
								Name: "bin/release",
								Body: fmt.Sprintf(`#!/bin/bash

  exit 1
`),
							},
						})
					})

					JustBeforeEach(func() {
						buildpacks = []builder.Buildpack{
							{
								Name:       "ruby_buildpack",
								Key:        "ruby_buildpack",
								URL:        urljoiner.Join(server.URL(), "/my-buildpack.zip"),
								SkipDetect: true,
							},
						}

						buildpackJSON, err = json.Marshal(buildpacks)
						Expect(err).ToNot(HaveOccurred())

						Expect(os.Setenv(eirinistaging.EnvBuildpacks, string(buildpackJSON))).To(Succeed())

						Expect(os.Setenv(eirinistaging.EnvCertsPath, certsPath)).To(Succeed())

						cmd := exec.Command(binaries.DownloaderPath)
						session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
						Eventually(session).Should(gexec.Exit())
						Expect(err).NotTo(HaveOccurred())
					})

					It("should fail with exit code 224", func() {
						eiriniServer.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.VerifyRequest("PUT", responseURL),
								verifyResponse(true, "BuildpackReleaseFailed: exit status 224"),
							),
						)

						cmd := exec.Command(binaries.ExecutorPath)
						session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
						Eventually(session, 80).Should(gexec.Exit())
						Expect(session.ExitCode()).To(Equal(builder.ReleaseFailCode))
					})
				})

			})
		})

		Context("with binary buildpack", func() {
			Context("when extract succeeds", func() {
				BeforeEach(func() {
					appbitBytes, err = ioutil.ReadFile("testdata/catnip.zip")
					Expect(err).NotTo(HaveOccurred())

					buildpackBytes, err = ioutil.ReadFile("testdata/binary-buildpack-cflinuxfs2-v1.0.32.zip")
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

					buildpacks = []builder.Buildpack{
						{
							Name:       "binary_buildpack",
							Key:        "binary_buildpack",
							URL:        urljoiner.Join(server.URL(), "/my-buildpack.zip"),
							SkipDetect: true,
						},
					}

					buildpackJSON, err = json.Marshal(buildpacks)
					Expect(err).ToNot(HaveOccurred())

					Expect(os.Setenv(eirinistaging.EnvBuildpacks, string(buildpackJSON))).To(Succeed())

					Expect(os.Setenv(eirinistaging.EnvCertsPath, certsPath)).To(Succeed())

					cmd := exec.Command(binaries.DownloaderPath)
					session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
					Eventually(session).Should(gexec.Exit())
					Expect(err).NotTo(HaveOccurred())

				})

				JustBeforeEach(func() {
					cmd := exec.Command(binaries.ExecutorPath)
					session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
					Eventually(session, 80).Should(gexec.Exit())
				})

				It("should create the droplet and output metadata", func() {
					Expect(session.ExitCode()).To(BeZero())

					Expect(path.Join(outputDir, "droplet.tgz")).To(BeARegularFile())
					Expect(path.Join(outputDir, "result.json")).To(BeARegularFile())
				})
			})
		})

		Context("upload", func() {
			BeforeEach(func() {
				appbitBytes, err = ioutil.ReadFile("testdata/dora.zip")
				Expect(err).NotTo(HaveOccurred())

				buildpackBytes, err = ioutil.ReadFile("testdata/ruby-buildpack-cflinuxfs2-v1.7.35.zip")
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

				Expect(os.Setenv(eirinistaging.EnvDropletUploadURL, urljoiner.Join(server.URL(), "my-droplet"))).To(Succeed())

				Expect(os.Setenv(eirinistaging.EnvDownloadURL, urljoiner.Join(server.URL(), "my-app-bits"))).To(Succeed())

				buildpacks = []builder.Buildpack{
					{
						Name: "ruby_buildpack",
						Key:  "ruby_buildpack",
						URL:  urljoiner.Join(server.URL(), "/my-buildpack.zip"),
					},
				}

				buildpackJSON, err = json.Marshal(buildpacks)
				Expect(err).ToNot(HaveOccurred())

				Expect(os.Setenv(eirinistaging.EnvBuildpacks, string(buildpackJSON))).To(Succeed())

				Expect(os.Setenv(eirinistaging.EnvCertsPath, certsPath)).To(Succeed())

				cmd := exec.Command(binaries.DownloaderPath)
				session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Eventually(session).Should(gexec.Exit())
				Expect(err).NotTo(HaveOccurred())

				cmd = exec.Command(binaries.ExecutorPath)
				session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Eventually(session, 80).Should(gexec.Exit())
				Expect(err).NotTo(HaveOccurred())
			})

			JustBeforeEach(func() {
				cmd := exec.Command(binaries.UploaderPath)
				session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Eventually(session, 10).Should(gexec.Exit())
			})

			It("should successfully upload the droplet", func() {
				Expect(session).To(gexec.Exit(0))
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
					Expect(session.ExitCode).NotTo(BeZero())
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

					Expect(session.ExitCode).NotTo(BeZero())
				})
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
