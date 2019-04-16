package packager_test

import (
	"github.com/cloudfoundry/cnb2cf/metadata"
	"github.com/cloudfoundry/cnb2cf/packager"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"gopkg.in/jarcoal/httpmock.v1"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

func TestUnitPackager(t *testing.T) {
	spec.Run(t, "UnitPackager", testUnitPackager, spec.Report(report.Terminal{}))
}

func testUnitPackager(t *testing.T, when spec.G, it spec.S) {
	var (
		tmpDir string
		err    error
	)

	it.Before(func() {
		RegisterTestingT(t)
		tmpDir, err = ioutil.TempDir("", "")
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		os.RemoveAll(tmpDir)
	})

	when("InstallCNBSource", func() {
		it.Before(func() {
			httpmock.Activate()
		})

		it.After(func() {
			httpmock.DeactivateAndReset()
		})

		it("installs the CNB source URI", func() {
			dep := metadata.V2Dependency{
				Name:         "some-cnb",
				Version:      "1.0.0",
				Source:       "https://example.com/cnb.tgz",
				SourceSHA256: "d1b2a59fbea7e20077af9f91b27e95e865061b270be03ff539ab3b73587882e8",
				CFStacks:     []string{"stack"},
			}

			httpmock.RegisterResponder("GET", "https://example.com/cnb.tgz",
				httpmock.NewStringResponder(200, "contents"))

			dstFile := filepath.Join(tmpDir, "some-cnb", "archive")
			Expect(packager.InstallCNBSource(dep, dstFile)).To(Succeed())
			Expect(dstFile).To(BeAnExistingFile())
		})
	})

	when("ExtractCNBSource", func() {
		it("gets the CNB source URI", func() {
			dep := metadata.V2Dependency{
				Source: "https://example.com/cnb.tgz",
			}

			tarPath := filepath.Join("testdata", "fake-file.tgz")

			Expect(packager.ExtractCNBSource(dep, tarPath, tmpDir)).To(Succeed())
			Expect(filepath.Join(tmpDir, "file")).To(BeAnExistingFile())
		})
	})

	when("UpdateManifest", func() {
		var (
			depPath string
			dep     metadata.V2Dependency
		)

		it.Before(func() {
			depPath, err = filepath.Abs(filepath.Join("testdata", "fake-file.tgz"))
			Expect(err).NotTo(HaveOccurred())
			dep = metadata.V2Dependency{
				Name:    "some-cnb",
				Version: "1.0.0",
				URI:     "https://example.com/cnb.tgz",
				SHA256:  "some-sha",
			}
		})

		it("creates an online cnb archive", func() {
			Expect(packager.UpdateDependency(&dep, depPath)).To(Succeed())

			Expect(dep.URI).To(Equal("file://" + depPath))
			Expect(dep.SHA256).To(Equal("84efae3d2c9ebecf21fe40ad397ba46ec8b0cc71155ac309f72e03f1347bc8e8"))
		})
	})

	when("FindCNB", func() {
		it("returns the path of the buildpack.toml if it is inside a directory", func() {
			sourcePath := filepath.Join("testdata", "cnb-source", "good-source")
			Expect(packager.FindCNB(sourcePath)).To(Equal(filepath.Join(sourcePath, "root-dir")))
		})

		it("returns the path of the buildpack.toml if it is top level", func() {
			sourcePath := filepath.Join("testdata", "cnb-source", "good-source-top-level")
			Expect(packager.FindCNB(sourcePath)).To(Equal(sourcePath))
		})

		it("returns the path of the buildpack.toml if it is top level", func() {
			sourcePath := filepath.Join("testdata", "cnb-source", "bad-source-no-toml")
			_, err := packager.FindCNB(sourcePath)
			Expect(err).To(MatchError("failed to find find cnb source: no buildpack.toml"))
		})

		it("returns error if there is no buildpack.toml", func() {
			sourcePath := filepath.Join("testdata", "cnb-source", "bad-source")
			_, err := packager.FindCNB(sourcePath)
			Expect(err).To(MatchError("failed to find find cnb source: found multiple buildpack.toml files"))
		})
	})

	when("CFPackage", func() {
		var (
			depPath string
			dep     metadata.V2Dependency
		)

		it.Before(func() {
			depPath, err = filepath.Abs(filepath.Join("testdata", "fake-file.tgz"))
			Expect(err).NotTo(HaveOccurred())
			dep = metadata.V2Dependency{
				Name:    "some-cnb",
				Version: "1.0.0",
				URI:     "https://example.com/cnb.tgz",
				SHA256:  "some-sha",
			}
		})

		it("creates an online cnb archive", func() {
			Expect(packager.UpdateDependency(&dep, depPath)).To(Succeed())

			Expect(dep.URI).To(Equal("file://" + depPath))
			Expect(dep.SHA256).To(Equal("84efae3d2c9ebecf21fe40ad397ba46ec8b0cc71155ac309f72e03f1347bc8e8"))
		})
	})
}
