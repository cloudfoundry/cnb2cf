package packager_test

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/cnb2cf/cloudnative"
	"github.com/cloudfoundry/cnb2cf/packager"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

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
	)

	it.Before(func() {
		Expect = NewWithT(t).Expect
		tmpDir, err = ioutil.TempDir("", "")
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		os.RemoveAll(tmpDir)
	})

	when("ExtractCNBSource", func() {
		it("extracts a tgz file", func() {
			dep := cloudnative.BuildpackMetadataDependency{
				Source: "https://example.com/cnb.tgz",
			}

			tarPath := filepath.Join("testdata", "fake-dir.tgz")

			Expect(packager.ExtractCNBSource(dep, tarPath, tmpDir)).To(Succeed())
			Expect(filepath.Join(tmpDir, "file")).To(BeAnExistingFile())
		})

		it("extracts a zip file", func() {
			dep := cloudnative.BuildpackMetadataDependency{
				Source: "https://example.com/cnb.zip",
			}

			tarPath := filepath.Join("testdata", "fake-dir.zip")

			Expect(packager.ExtractCNBSource(dep, tarPath, tmpDir)).To(Succeed())
			Expect(filepath.Join(tmpDir, "file")).To(BeAnExistingFile())
		})

		it("copies a directory", func() {
			dep := cloudnative.BuildpackMetadataDependency{
				Source: "file:///tmp/foo/",
			}

			tarPath := filepath.Join("testdata", "fake-dir/")

			Expect(packager.ExtractCNBSource(dep, tarPath, tmpDir)).To(Succeed())
			Expect(filepath.Join(tmpDir, "file")).To(BeAnExistingFile())
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

		it("returns error when there is no buildpack.toml ", func() {
			sourcePath := filepath.Join("testdata", "cnb-source", "bad-source-no-toml")
			_, err := packager.FindCNB(sourcePath)
			Expect(err).To(MatchError("failed to find find cnb source: no buildpack.toml"))
		})

		it("returns error  when there are multiple cnbs", func() {
			sourcePath := filepath.Join("testdata", "cnb-source", "bad-source-multiple-cnbs")
			_, err := packager.FindCNB(sourcePath)
			Expect(err).To(MatchError("failed to find find cnb source: found multiple buildpack.toml files"))
		})
	})
	when("BuildCNB", func() {
		it("returns buildpack tgz and sha", func() {
			sourcePath := filepath.Join("testdata", "cnb-source", "packagable")
			badFileName := filepath.Join(tmpDir, "paketo-buildpacks_node-engine")
			tarPath, sha, err := packager.BuildCNB(sourcePath, badFileName, true, "1.2.3")
			Expect(err).NotTo(HaveOccurred())
			Expect(tarPath).To(BeAnExistingFile())

			f, err := os.Open(tarPath)
			Expect(err).NotTo(HaveOccurred())
			defer f.Close()
			h := sha256.New()
			_, err = io.Copy(h, f)
			Expect(err).NotTo(HaveOccurred())
			Expect(sha).To(Equal(fmt.Sprintf("%x", h.Sum(nil))))
		})

		it("returns error when outputDir is an invalid file name", func() {
			sourcePath := filepath.Join("testdata", "cnb-source", "packagable")
			badFileName := filepath.Join(tmpDir, "paketo-buildpacks/node-engine")
			_, _, err := packager.BuildCNB(sourcePath, badFileName, true, "1.2.3")
			Expect(err).To(MatchError(ContainSubstring("invalid outputDir")))
		})
	})
}
