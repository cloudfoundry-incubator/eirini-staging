package eirinistaging_test

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/bbs/models"
	. "code.cloudfoundry.org/eirini-staging"
	"code.cloudfoundry.org/runtimeschema/cc_messages"
	"code.cloudfoundry.org/tlsconfig"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Responder", func() {
	Context("when responding to cc-uploader", func() {
		var (
			server    *ghttp.Server
			responder Responder
		)

		BeforeEach(func() {
			server = ghttp.NewUnstartedServer()
			certsPath, err := filepath.Abs("integration/testdata/certs")
			Expect(err).NotTo(HaveOccurred())

			certPath := filepath.Join(certsPath, "eirini.crt")
			keyPath := filepath.Join(certsPath, "eirini.key")
			caCertPath := filepath.Join(certsPath, CACertName)

			tlsConfig, err := tlsconfig.Build(
				tlsconfig.WithInternalServiceDefaults(),
				tlsconfig.WithIdentityFromFile(certPath, keyPath),
			).Server(
				tlsconfig.WithClientAuthenticationFromFile(caCertPath),
			)
			Expect(err).NotTo(HaveOccurred())
			server.HTTPTestServer.TLS = tlsConfig
			server.HTTPTestServer.StartTLS()

			stagingGUID := "staging-guid"
			completionCallback := "completion-call-me-back"
			eiriniCACertPath := filepath.Join(certsPath, CACertName)
			eiriniClientCert := filepath.Join(certsPath, EiriniClientCert)
			clientKey := filepath.Join(certsPath, EiriniClientKey)
			eiriniAddr := server.URL()

			responder, _ = NewResponder(stagingGUID, completionCallback, eiriniAddr, eiriniCACertPath, eiriniClientCert, clientKey)
		})

		AfterEach(func() {
			server.Close()
		})

		Context("when provided tls certs are missing", func() {

			It("should return an error", func() {
				_, initErr := NewResponder("guid", "callback", "0.0.0.0:1", "does-not-exist", "does-not-exist", "does-not-exist")
				Expect(initErr).To(MatchError(ContainSubstring("failed to create http client")))
			})
		})

		Context("when the provided certificates are not valid for the server", func() {
			BeforeEach(func() {
				server.RouteToHandler("PUT", "/stage/staging-guid/completed",
					ghttp.VerifyJSON(`{
						"task_guid": "staging-guid",
						"failed": false,
						"failure_reason": "",
						"result": "",
						"created_at": 0
					}`),
				)
			})

			It("should return an error", func() {
				resp := models.TaskCallbackResponse{
					TaskGuid: "staging-guid",
				}

				stagingGUID := "staging-guid"
				completionCallback := "completion-call-me-back"

				certsPath, err := filepath.Abs("integration/testdata/certs")
				Expect(err).ToNot(HaveOccurred())
				eiriniCACertPath := filepath.Join(certsPath, "internal-ca-cert")
				eiriniClientCert := filepath.Join(certsPath, "not-exactly-valid.crt")
				clientKey := filepath.Join(certsPath, "not-exactly-valid.key")
				eiriniAddr := server.URL()

				responder, _ = NewResponder(stagingGUID, completionCallback, eiriniAddr, eiriniCACertPath, eiriniClientCert, clientKey)
				err = responder.RespondWithSuccess(&resp)
				Expect(err).To(MatchError(ContainSubstring("request failed")))
			})

		})

		Context("when there is an error", func() {
			BeforeEach(func() {
				server.RouteToHandler("PUT", "/stage/staging-guid/completed",
					ghttp.VerifyJSON(`{
						"task_guid": "staging-guid",
						"failed": true,
						"failure_reason": "sploded",
						"result": "",
						"annotation": "{\"lifecycle\":\"\",\"completion_callback\":\"completion-call-me-back\"}",
						"created_at": 0
					}`),
				)
			})

			It("should respond with failure", func() {
				responder.RespondWithFailure(errors.New("sploded"))
			})
		})

		Context("when the response is success", func() {

			var (
				resultsFilePath string
				resultContents  string
				buildpacks      []byte
			)

			Context("when preparing the response results", func() {
				Context("when the results file is missing", func() {
					It("should error with missing file msg", func() {
						_, err := responder.PrepareSuccessResponse(resultsFilePath, string(buildpacks))
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("failed to read result.json"))
					})
				})

				Context("when the results json file is invalid", func() {
					It("should error when unmarhsaling the content", func() {
						resultsFilePath = resultsFile(resultContents)
						buildpack := cc_messages.Buildpack{}
						buildpacks, err := json.Marshal([]cc_messages.Buildpack{buildpack})
						Expect(err).NotTo(HaveOccurred())

						_, err = responder.PrepareSuccessResponse(resultsFilePath, string(buildpacks))
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("unexpected end of JSON input"))
					})
				})

			})

			Context("when response preparation is successful", func() {
				BeforeEach(func() {
					resultContents = `{"lifecycle_type":"no-type", "execution_metadata":"data"}`
					server.RouteToHandler("PUT", "/stage/staging-guid/completed",
						ghttp.VerifyJSON(`{
						"task_guid": "staging-guid",
						"failed": false,
						"failure_reason": "",
						"result": "{\"lifecycle_metadata\":{\"detected_buildpack\":\"\",\"buildpacks\":null},\"process_types\":null,\"execution_metadata\":\"data\",\"lifecycle_type\":\"no-type\"}",
						"annotation": "{\"lifecycle\":\"\",\"completion_callback\":\"completion-call-me-back\"}",
						"created_at": 0
					}`),
					)

					resultsFilePath = resultsFile(resultContents)

				})

				AfterEach(func() {
					Expect(os.Remove(resultsFilePath)).To(Succeed())
				})

				It("should respond with failure", func() {
					buildpack := cc_messages.Buildpack{}
					buildpacks, err := json.Marshal([]cc_messages.Buildpack{buildpack})
					Expect(err).NotTo(HaveOccurred())

					resp, err := responder.PrepareSuccessResponse(resultsFilePath, string(buildpacks))
					Expect(err).NotTo(HaveOccurred())
					err = responder.RespondWithSuccess(resp)
					Expect(err).NotTo(HaveOccurred())
				})
			})
		})

	})
})

func resultsFile(content string) string {

	tmpfile, err := ioutil.TempFile("", "metadata_result")
	Expect(err).ToNot(HaveOccurred())

	_, err = tmpfile.Write([]byte(content))
	Expect(err).ToNot(HaveOccurred())

	err = tmpfile.Close()
	Expect(err).ToNot(HaveOccurred())

	return tmpfile.Name()
}
