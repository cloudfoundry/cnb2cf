package shims_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/cnb2cf/shims"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testReleaser(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect func(interface{}, ...interface{}) Assertion

		releaser shims.Releaser
		v2AppDir string
		buf      *bytes.Buffer
	)

	it.Before(func() {
		Expect = NewWithT(t).Expect

		var err error
		v2AppDir, err = ioutil.TempDir("", "build")
		Expect(err).NotTo(HaveOccurred())

		contents := `
				buildpacks = ["some.buildpacks", "some.other.buildpack"]
				[[processes]]
				type = "web"
				command = "npm start"
				`
		Expect(os.MkdirAll(filepath.Join(v2AppDir, ".cloudfoundry"), 0777)).To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(v2AppDir, ".cloudfoundry", "metadata.toml"), []byte(contents), 0666)).To(Succeed())

		buf = &bytes.Buffer{}

		releaser = shims.Releaser{
			MetadataPath: filepath.Join(v2AppDir, ".cloudfoundry", "metadata.toml"),
			Writer:       buf,
		}
	})

	it.After(func() {
		Expect(os.RemoveAll(v2AppDir)).To(Succeed())
	})

	it("runs with the correct arguments and moves things to the correct place", func() {
		Expect(releaser.Release()).To(Succeed())
		Expect(buf.Bytes()).To(Equal([]byte("default_process_types:\n  web: npm start\n")))
		Expect(filepath.Join(v2AppDir, ".cloudfoundry", "metadata.toml")).NotTo(BeAnExistingFile())
	})
}
