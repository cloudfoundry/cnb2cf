package packager_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/cnb2cf/metadata"
	"github.com/cloudfoundry/cnb2cf/packager"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	httpmock "gopkg.in/jarcoal/httpmock.v1"

	. "github.com/onsi/gomega"
)

func TestUnitPackager(t *testing.T) {
	spec.Run(t, "UnitPackager", testUnitPackager, spec.Report(report.Terminal{}))
}

func testUnitPackager(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect func(interface{}, ...interface{}) Assertion
		tmpDir string
		err    error
		p      packager.Packager
	)

	it.Before(func() {
		Expect = NewWithT(t).Expect
		tmpDir, err = ioutil.TempDir("", "")
		Expect(err).NotTo(HaveOccurred())
		p = packager.Packager{Dev: false}
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

		it("installs the remote CNB source URI even if dev flag set", func() {
			p.Dev = true
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
			Expect(p.InstallCNBSource(dep, dstFile)).To(Succeed())
			Expect(dstFile).To(BeAnExistingFile())
		})

		it("copies if the CNB source is a dir", func() {
			p.Dev = true

			dep := metadata.V2Dependency{
				Name:         "some-cnb",
				Version:      "1.0.0",
				Source:       "testdata/fake-dir/",
				SourceSHA256: "",
				CFStacks:     []string{"stack"},
			}

			dstFile := filepath.Join(tmpDir, "some-dst")
			Expect(p.InstallCNBSource(dep, dstFile)).To(Succeed())
			Expect(dstFile).To(BeADirectory())
			Expect(filepath.Join(dstFile, "file")).To(BeAnExistingFile())
		})
	})

	when("ExtractCNBSource", func() {
		it("extracts a tgz file", func() {
			dep := metadata.V2Dependency{
				Source: "https://example.com/cnb.tgz",
			}

			tarPath := filepath.Join("testdata", "fake-dir.tgz")

			Expect(p.ExtractCNBSource(dep, tarPath, tmpDir)).To(Succeed())
			Expect(filepath.Join(tmpDir, "file")).To(BeAnExistingFile())
		})

		it("extracts a zip file", func() {
			dep := metadata.V2Dependency{
				Source: "https://example.com/cnb.zip",
			}

			tarPath := filepath.Join("testdata", "fake-dir.zip")

			Expect(p.ExtractCNBSource(dep, tarPath, tmpDir)).To(Succeed())
			Expect(filepath.Join(tmpDir, "file")).To(BeAnExistingFile())
		})

		it("copies a directory", func() {
			dep := metadata.V2Dependency{
				Source: "file:///tmp/foo/",
			}

			tarPath := filepath.Join("testdata", "fake-dir/")

			Expect(p.ExtractCNBSource(dep, tarPath, tmpDir)).To(Succeed())
			Expect(filepath.Join(tmpDir, "file")).To(BeAnExistingFile())
		})
	})

	when("UpdateManifest", func() {
		var (
			depPath string
			dep     metadata.V2Dependency
		)

		it.Before(func() {
			depPath, err = filepath.Abs(filepath.Join("testdata", "fake-dir.tgz"))
			Expect(err).NotTo(HaveOccurred())
			dep = metadata.V2Dependency{
				Name:    "some-cnb",
				Version: "1.0.0",
				URI:     "https://example.com/cnb.tgz",
				SHA256:  "some-sha",
			}
		})

		it("creates an online cnb archive", func() {
			Expect(p.UpdateDependency(&dep, depPath)).To(Succeed())

			Expect(dep.URI).To(Equal("file://" + depPath))
			Expect(dep.SHA256).To(Equal("84efae3d2c9ebecf21fe40ad397ba46ec8b0cc71155ac309f72e03f1347bc8e8"))
		})
	})

	when("FindCNB", func() {
		it("returns the path of the buildpack.toml if it is inside a directory", func() {
			sourcePath := filepath.Join("testdata", "cnb-source", "good-source")
			Expect(p.FindCNB(sourcePath)).To(Equal(filepath.Join(sourcePath, "root-dir")))
		})

		it("returns the path of the buildpack.toml if it is top level", func() {
			sourcePath := filepath.Join("testdata", "cnb-source", "good-source-top-level")
			Expect(p.FindCNB(sourcePath)).To(Equal(sourcePath))
		})

		it("returns error when there is no buildpack.toml ", func() {
			sourcePath := filepath.Join("testdata", "cnb-source", "bad-source-no-toml")
			_, err := p.FindCNB(sourcePath)
			Expect(err).To(MatchError("failed to find find cnb source: no buildpack.toml"))
		})

		it("returns error  when there are multiple cnbs", func() {
			sourcePath := filepath.Join("testdata", "cnb-source", "bad-source-multiple-cnbs")
			_, err := p.FindCNB(sourcePath)
			Expect(err).To(MatchError("failed to find find cnb source: found multiple buildpack.toml files"))
		})
	})

	when("CFPackage", func() {
		var (
			depPath string
			dep     metadata.V2Dependency
		)

		it.Before(func() {
			depPath, err = filepath.Abs(filepath.Join("testdata", "fake-dir.tgz"))
			Expect(err).NotTo(HaveOccurred())
			dep = metadata.V2Dependency{
				Name:    "some-cnb",
				Version: "1.0.0",
				URI:     "https://example.com/cnb.tgz",
				SHA256:  "some-sha",
			}
		})

		it("creates an online cnb archive", func() {
			Expect(p.UpdateDependency(&dep, depPath)).To(Succeed())

			Expect(dep.URI).To(Equal("file://" + depPath))
			Expect(dep.SHA256).To(Equal("84efae3d2c9ebecf21fe40ad397ba46ec8b0cc71155ac309f72e03f1347bc8e8"))
		})
	})
}
