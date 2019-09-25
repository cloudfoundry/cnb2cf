package cloudnative_test

import (
	"io/ioutil"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/cnb2cf/cloudnative"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testFilesystem(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect func(interface{}, ...interface{}) Assertion

		tmpDir     string
		filesystem cloudnative.Filesystem
	)

	it.Before(func() {
		Expect = NewWithT(t).Expect

		var err error
		tmpDir, err = ioutil.TempDir("", "dependency-installer")
		Expect(err).NotTo(HaveOccurred())

		err = ioutil.WriteFile(filepath.Join(tmpDir, "some-file"), []byte("some-file-content"), 0644)
		Expect(err).NotTo(HaveOccurred())

		filesystem = cloudnative.NewFilesystem(http.Dir(tmpDir))
	})

	when("ReadFile", func() {
		it("returns the contents of the file as bytes", func() {
			content, err := filesystem.ReadFile("some-file")
			Expect(err).NotTo(HaveOccurred())
			Expect(content).To(Equal([]byte("some-file-content")))
		})
	})
}
