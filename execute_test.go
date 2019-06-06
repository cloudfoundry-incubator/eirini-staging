package eirinistaging_test

import (
	"errors"
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	bap "code.cloudfoundry.org/buildpackapplifecycle"
	. "code.cloudfoundry.org/eirini-staging"
	"code.cloudfoundry.org/eirini-staging/builder"
	"code.cloudfoundry.org/eirini-staging/eirinistagingfakes"
)

const (
	downloadDir = "some-dir"
)

var _ = Describe("PacksExecutor", func() {

	var (
		executor       Executor
		extractor      *eirinistagingfakes.FakeExtractor
		commander      *eirinistagingfakes.FakeCommander
		resultModifier *eirinistagingfakes.FakeStagingResultModifier
		tmpfile        *os.File
		resultContents string
	)

	createTmpFile := func() {
		var err error

		tmpfile, err = ioutil.TempFile("", "metadata_result")
		Expect(err).ToNot(HaveOccurred())

		_, err = tmpfile.Write([]byte(resultContents))
		Expect(err).ToNot(HaveOccurred())

		err = tmpfile.Close()
		Expect(err).ToNot(HaveOccurred())
	}

	BeforeEach(func() {
		commander = new(eirinistagingfakes.FakeCommander)
		resultModifier = new(eirinistagingfakes.FakeStagingResultModifier)
		extractor = new(eirinistagingfakes.FakeExtractor)

		resultModifier.ModifyStub = func(result bap.StagingResult) (bap.StagingResult, error) {
			return result, nil
		}

		resultContents = `{"lifecycle_type":"no-type", "execution_metadata":"data"}`
	})

	JustBeforeEach(func() {
		createTmpFile()
		packsConf := builder.Config{
			PacksBuilderPath:          "/packs/builder",
			BuildpacksDir:             "/var/lib/buildpacks",
			OutputDropletLocation:     "/out/droplet.tgz",
			OutputBuildArtifactsCache: "/cache/cache.tgz",
			OutputMetadataLocation:    tmpfile.Name(),
		}

		executor = &PacksExecutor{
			Conf: &packsConf,
			// DownloadDir: downloadDir,
		}

	})

	AfterEach(func() {
		Expect(os.Remove(tmpfile.Name())).To(Succeed())
	})

	Context("When executing a recipe", func() {

		var (
			err error
		)

		Context("when extracting fails", func() {
			JustBeforeEach(func() {
				err = executor.ExecuteRecipe()
			})

			BeforeEach(func() {
				extractor.ExtractReturns(errors.New("some-error"))
			})

			It("should fail to execute", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("some-error"))
			})
		})

		Context("when buildpack information is not set", func() {
			JustBeforeEach(func() {
				err = executor.ExecuteRecipe()
			})

			It("should not return an error", func() {
				Expect(err).ToNot(HaveOccurred())
			})

			It("should call the extractor", func() {
				downloadPath, actualTargetDir := extractor.ExtractArgsForCall(0)
				Expect(extractor.ExtractCallCount()).To(Equal(1))
				Expect(downloadPath).To(ContainSubstring(downloadDir))
				Expect(actualTargetDir).NotTo(BeEmpty())
			})

			It("should run the packs builder", func() {
				Expect(commander.ExecCallCount()).To(Equal(1))
				_, actualTargetDir := extractor.ExtractArgsForCall(0)

				cmd, args := commander.ExecArgsForCall(0)
				Expect(cmd).To(Equal("/packs/builder"))
				Expect(args).To(ConsistOf(
					"-buildDir", actualTargetDir,
					"-buildpacksDir", "/var/lib/buildpacks",
					"-outputDroplet", "/out/droplet.tgz",
					"-outputBuildArtifactsCache", "/cache/cache.tgz",
					"-outputMetadata", tmpfile.Name(),
				))
			})
		})

		Context("when single buildpack information is provided", func() {
			JustBeforeEach(func() {
				typedExecutor := executor.(*PacksExecutor)
				typedExecutor.BuildpacksJSON = `
					[
						{
							"name":"binary_buildpack",
							"key":"binary_buildpack_key",
							"url":"https://example.com/binary_buildpack",
							"skip_detect": true
						}
					]`
				err = typedExecutor.ExecuteRecipe()
			})

			It("should not return an error", func() {
				Expect(err).ToNot(HaveOccurred())
			})

			It("should call the extractor", func() {
				downloadPath, actualTargetDir := extractor.ExtractArgsForCall(0)
				Expect(extractor.ExtractCallCount()).To(Equal(1))
				Expect(downloadPath).To(ContainSubstring(downloadDir))
				Expect(actualTargetDir).NotTo(BeEmpty())
			})

			It("should run the packs builder", func() {
				Expect(commander.ExecCallCount()).To(Equal(1))
				_, actualTargetDir := extractor.ExtractArgsForCall(0)

				cmd, args := commander.ExecArgsForCall(0)
				Expect(cmd).To(Equal("/packs/builder"))
				Expect(args).To(ConsistOf(
					"-buildDir", actualTargetDir,
					"-buildpacksDir", "/var/lib/buildpacks",
					"-outputDroplet", "/out/droplet.tgz",
					"-outputBuildArtifactsCache", "/cache/cache.tgz",
					"-outputMetadata", tmpfile.Name(),
					"-skipDetect",
					"-buildpackOrder", "binary_buildpack",
				))
			})
		})
	})
})
