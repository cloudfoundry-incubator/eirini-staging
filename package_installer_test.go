package eirinistaging_test

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	. "code.cloudfoundry.org/eirini-staging"
	eirinistagingfakes "code.cloudfoundry.org/eirini-staging/eirini-stagingfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

//go:generate counterfeiter io.Reader

var _ = Describe("PackageInstaller", func() {
	var (
		err           error
		downloadURL   string
		downloadDir   string
		installer     Installer
		server        *ghttp.Server
		zippedPackage []byte
		readerFrom    ReaderFrom
	)

	BeforeEach(func() {
		readerFrom = nil
		zippedPackage, err = makeZippedPackage()
		Expect(err).ToNot(HaveOccurred())

		server = ghttp.NewServer()
		server.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/some-app-guid"),
				ghttp.RespondWith(http.StatusOK, zippedPackage),
			),
		)
		downloadURL = server.URL() + "/some-app-guid"

		downloadDir, err = ioutil.TempDir("", "downloadDir")
		Expect(err).ToNot(HaveOccurred())
	})

	JustBeforeEach(func() {
		installer = NewPackageManager(&http.Client{}, downloadURL, downloadDir, readerFrom)
		err = installer.Install()
	})

	AfterEach(func() {
		server.Close()
	})

	Context("package is installed successfully", func() {
		It("succeeds", func() {
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("When an empty downloadURL is provided", func() {
		BeforeEach(func() {
			downloadURL = ""
		})

		It("should return an error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("empty downloadURL provided")))
		})
	})

	Context("When an empty downloadDir is provided", func() {
		BeforeEach(func() {
			downloadDir = ""
		})

		It("should return an error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("empty downloadDir provided")))
		})
	})

	Context("When the download fails", func() {
		Context("When the http server returns an error code", func() {
			BeforeEach(func() {
				server.Close()
				server = ghttp.NewUnstartedServer()
			})

			It("should error with an corresponding error message", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("failed to perform get request")))
			})
		})

		Context("When the server does not return OK HTTP status", func() {
			BeforeEach(func() {
				server.RouteToHandler("GET", "/some-app-guid",
					ghttp.RespondWith(http.StatusTeapot, nil),
				)
			})

			It("should return an meaningful err message", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("download failed. status code")))
			})
		})

		Context("When the app id creates an invalid URL", func() {
			BeforeEach(func() {
				downloadURL = "%&"
			})

			It("should return an error", func() {
				Expect(err).To(HaveOccurred())
			})

			It("should return the right error message", func() {
				Expect(err).To(MatchError(ContainSubstring("failed to perform get request")))
				Expect(err).To(MatchError(ContainSubstring(downloadURL)))
			})
		})

		Context("when a custom response reader is used", func() {
			var fakeReader *eirinistagingfakes.FakeReader

			BeforeEach(func() {
				fakeReader = new(eirinistagingfakes.FakeReader)
				fakeReader.ReadReturns(0, io.EOF)
				readerFrom = func(io.Reader) io.Reader {
					return fakeReader
				}
			})

			It("uses the custom reader", func() {
				Expect(fakeReader.ReadCallCount()).To(Equal(1))
			})
		})
	})
})

// straight from https://golang.org/pkg/archive/zip/#example_Writer
func makeZippedPackage() ([]byte, error) {
	buf := bytes.Buffer{}
	w := zip.NewWriter(&buf)

	// the ZIP file is intentionally left empty

	err := w.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close zip writer: %w", err)
	}

	return buf.Bytes(), nil
}
