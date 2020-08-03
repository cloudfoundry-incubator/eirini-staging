package checksum_test

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"strings"

	. "code.cloudfoundry.org/eirini-staging/checksum"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Verifying Reader", func() {
	const loremIpsum string = "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua."

	var (
		checksum    string
		outputBytes []byte
		readErr     error
	)

	BeforeEach(func() {
		loremIpsumChecksum := sha256.New()
		_, err := loremIpsumChecksum.Write([]byte(loremIpsum))
		Expect(err).NotTo(HaveOccurred())
		checksum = fmt.Sprintf("%x", loremIpsumChecksum.Sum(nil))
	})

	JustBeforeEach(func() {
		reader := NewVerifyingReader(strings.NewReader(loremIpsum), sha256.New(), checksum)
		outputBytes, readErr = ioutil.ReadAll(reader)
	})

	It("succeeds", func() {
		Expect(readErr).NotTo(HaveOccurred())
	})

	It("reads the string", func() {
		Expect(string(outputBytes)).To(Equal(loremIpsum))
	})

	When("the checksum cannot be verified", func() {
		BeforeEach(func() {
			checksum = "cant-verify-this"
		})

		It("returns an error", func() {
			Expect(readErr).To(MatchError(ContainSubstring("checksum")))
		})
	})
})
