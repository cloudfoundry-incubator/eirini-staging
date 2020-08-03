package builder_test

import (
	"code.cloudfoundry.org/eirini-staging/builder"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {

	var config builder.Config

	BeforeEach(func() {
		config = builder.Config{}
	})

	Describe("Init buildpacks", func() {

		var buildpackJSON string

		BeforeEach(func() {
			buildpackJSON = `
			[{
				"name": "java_buildpack",
				"skip_detect": false
			}, {
				"name": "ruby_buildpack",
				"skip_detect": false
			}, {
				"name": "nodejs_buildpack",
				"skip_detect": true
			}]`
		})

		JustBeforeEach(func() {
			Expect(config.InitBuildpacks(buildpackJSON)).To(Succeed())
		})

		It("should set the correct buildpack order", func() {
			Expect(config.BuildpackOrder).To(ConsistOf(
				"java_buildpack",
				"ruby_buildpack",
				"nodejs_buildpack",
			))
		})

		It("should not skip detect", func() {
			Expect(config.SkipDetect).To(BeFalse())
		})

		When("buildpack detection is skipped", func() {
			BeforeEach(func() {
				buildpackJSON = `
				[{
					"name": "java_buildpack",
					"skip_detect": true
				}, {
					"name": "ruby_buildpack",
					"skip_detect": true
				}]`
			})

			It("should skip detect", func() {
				Expect(config.SkipDetect).To(BeTrue())
			})
		})

		When("buildpacks json is empty", func() {

			BeforeEach(func() {
				buildpackJSON = ""
			})

			It("should not change the defaults", func() {
				Expect(config.BuildpackOrder).To(BeEmpty())
				Expect(config.SkipDetect).To(BeFalse())
			})
		})
	})

})
