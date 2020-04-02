package shims_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/cnb2cf/shims"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testShimUtils(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect func(interface{}, ...interface{}) Assertion
		tmpDir string
		env    []string
	)

	it.Before(func() {
		var err error
		Expect = NewWithT(t).Expect
		tmpDir, err = ioutil.TempDir("", "WritePlatformDir")
		Expect(err).NotTo(HaveOccurred())
		env = []string{"key1=value1", "key2=value=2"}
	})

	when("WritePlatformDir", func() {
		it("writes the env to files", func() {
			Expect(shims.WritePlatformDir(tmpDir, env)).To(Succeed())

			envDir := filepath.Join(tmpDir, "env")
			envFiles, err := filepath.Glob(filepath.Join(envDir, "*"))
			Expect(err).NotTo(HaveOccurred())
			Expect(envFiles).To(HaveLen(2))

			key1File := filepath.Join(envDir, "key1")
			Expect(filepath.Join(key1File)).To(BeARegularFile())
			contents, err := ioutil.ReadFile(key1File)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(Equal("value1"))

			key2File := filepath.Join(envDir, "key2")
			Expect(filepath.Join(key2File)).To(BeARegularFile())
			contents, err = ioutil.ReadFile(key2File)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(Equal("value=2"))
		})
	})

	when("error cases", func() {
		when("when unable to make env dir", func() {
			it.Before(func() {
				Expect(os.Chmod(tmpDir, 0000)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(tmpDir, os.ModePerm)).To(Succeed())
			})

			it("returns an error", func() {
				Expect(shims.WritePlatformDir(tmpDir, env)).To(MatchError(
					ContainSubstring("unable to make env dir:")))
			})
		})

		when("when unable to write env files", func() {
			var envPath string

			it.Before(func() {
				envPath = filepath.Join(tmpDir, "env")
				os.MkdirAll(envPath, os.ModePerm)
				Expect(os.Chmod(envPath, 0000)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(envPath, os.ModePerm)).To(Succeed())
			})

			it("returns an error", func() {
				Expect(shims.WritePlatformDir(tmpDir, env)).To(MatchError(
					ContainSubstring("unable to write key1 env file:")))
			})
		})
		when("env has invalid pairing", func() {
			it("returns an error", func() {
				Expect(shims.WritePlatformDir(tmpDir, []string{"no-key-val"})).To(MatchError(
					ContainSubstring("var fails to contain required key=value structure")))
			})
		})
	})
}
