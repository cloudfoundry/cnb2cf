package shimmer_test

import (
	"archive/zip"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/cnb2cf/shimmer"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestPackagerUnit(t *testing.T) {
	spec.Run(t, "PackagerUnit", testPackagerUnit, spec.Report(report.Terminal{}))
}

func testPackagerUnit(t *testing.T, when spec.G, it spec.S) {
	when("packaging", func() {
		var (
			bpDir, tempDir string
			config         shimmer.Config
			err            error
		)

		it.Before(func() {
			RegisterTestingT(t)

			tempDir, err = ioutil.TempDir("", "")
			Expect(err).NotTo(HaveOccurred())

			config = shimmer.Config{
				Language: "some-language",
				Version:  "1.1.1",
				Stack:    "cflinuxfs3",
			}

			bpDir = "testdata"
		})

		it.After(func() {
			os.RemoveAll(tempDir)
		})

		it("creates the buildpack zip", func() {
			Expect(shimmer.CreateZip(config, bpDir, tempDir)).To(Succeed())
			r, err := zip.OpenReader(filepath.Join(tempDir, "some-language_buildpack-cflinuxfs3-1.1.1.zip"))
			Expect(err).NotTo(HaveOccurred())

			defer r.Close()

			fileNames := []string{}
			for _, f := range r.File {
				fileNames = append(fileNames, f.Name)
			}

			Expect(fileNames).To(ConsistOf([]string{
				"VERSION",
				"bin/compile",
				"bin/detect",
				"bin/finalize",
				"bin/release",
				"bin/supply",
				"manifest.yml",
				"order.toml",
			}))
		})
	}, spec.Random())
}
