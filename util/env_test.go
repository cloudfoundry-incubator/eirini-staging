package util_test

import (
	"os"

	"code.cloudfoundry.org/eirini-staging/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Env", func() {
	const neverUsedEnvVar string = "FOO_VAR_SHOULD_BE_NEVER_USED"

	Describe("GetEnvOrDefault", func() {
		BeforeEach(func() {
			_, ok := os.LookupEnv(neverUsedEnvVar)
			Expect(ok).To(BeFalse())
		})

		It("returns the default value for env vars that are not set", func() {
			Expect(util.GetEnvOrDefault("FOO_VAR_SHOULD_BE_NEVER_USED", "default")).To(Equal("default"))
		})

		When("the environment variable is set", func() {
			BeforeEach(func() {
				Expect(os.Setenv(neverUsedEnvVar, "neverland")).To(Succeed())
			})

			AfterEach(func() {
				Expect(os.Unsetenv(neverUsedEnvVar)).To(Succeed())
			})

			It("returns the var value", func() {
				Expect(util.GetEnvOrDefault(neverUsedEnvVar, "abcd")).To(Equal("neverland"))
			})
		})
	})

	Describe("MustGetEnv", func() {
		BeforeEach(func() {
			Expect(os.Setenv(neverUsedEnvVar, "void")).To(Succeed())
		})

		AfterEach(func() {
			Expect(os.Unsetenv(neverUsedEnvVar)).To(Succeed())
		})

		It("returns the var value", func() {
			Expect(util.MustGetEnv(neverUsedEnvVar)).To(Equal("void"))
		})
	})
})
