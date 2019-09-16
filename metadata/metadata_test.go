package metadata_test

import (
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/libcfbuildpack/buildpack"

	"github.com/cloudfoundry/cnb2cf/metadata"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

func TestUnitMetadata(t *testing.T) {
	spec.Run(t, "UnitMetadata", testUnitMetadata, spec.Report(report.Terminal{}))
}

func testUnitMetadata(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect func(interface{}, ...interface{}) Assertion
	)

	it.Before(func() {
		Expect = NewWithT(t).Expect
	})

	when("UpdateManifest", func() {
		var (
			depPath string
			dep     metadata.Dependency
			err     error
		)

		it.Before(func() {
			depPath, err = filepath.Abs(filepath.Join("testdata", "fake-dir.tgz"))
			Expect(err).NotTo(HaveOccurred())
			dep = metadata.Dependency{
				Name:    "some-cnb",
				Version: "1.0.0",
				URI:     "https://example.com/cnb.tgz",
				SHA256:  "some-sha",
			}
		})

		it("creates an online cnb archive", func() {
			Expect(dep.UpdateDependency(depPath)).To(Succeed())

			Expect(dep.URI).To(Equal("file://" + depPath))
			Expect(dep.SHA256).To(Equal("84efae3d2c9ebecf21fe40ad397ba46ec8b0cc71155ac309f72e03f1347bc8e8"))
		})
	})

	when("BuildpackToml", func() {
		when("validating buildpack.toml", func() {
			it("Passes Validation on valid buildpack.toml", func() {
				obp := metadata.BuildpackToml{}
				Expect(obp.Load(filepath.Join("testdata", "valid-obp.toml"))).To(Succeed())
			})

			it("fails on empty buildpack.toml", func() {
				obp := metadata.BuildpackToml{}
				Expect(obp.Validate()).NotTo(Succeed())
			})

			it("fails without dependencies and order", func() {
				obp := metadata.BuildpackToml{
					Info: buildpack.Info{
						ID:      "some-id",
						Version: "some-version",
					},
				}
				Expect(obp.Validate()).NotTo(Succeed())

				obp = metadata.BuildpackToml{
					Info: buildpack.Info{
						ID:      "some-id",
						Version: "some-version",
					},
					Order: []metadata.Order{{Group: []metadata.CNBBuildpack{
						{
							ID:      "2",
							Version: "3",
						}}}},
				}
				Expect(obp.Validate()).NotTo(Succeed())

			})

			it("Fails Validation on invalid order in  buildpack.toml", func() {
				obp := metadata.BuildpackToml{}
				Expect(obp.Load(filepath.Join("testdata", "invalid-order-obp.toml"))).NotTo(Succeed())
			})
			it("Fails Validation on invalid dependency in buildpack.toml", func() {
				obp := metadata.BuildpackToml{}
				Expect(obp.Load(filepath.Join("testdata", "invalid-dep-obp.toml"))).NotTo(Succeed())
			})
		})
	})
}
