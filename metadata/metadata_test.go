package metadata_test

import (
	"path/filepath"
	"testing"

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

	when("UpdateDependency", func() {
		var (
			depPath string
			err     error
		)

		it.Before(func() {
			depPath, err = filepath.Abs(filepath.Join("testdata", "fake-dir.tgz"))
			Expect(err).NotTo(HaveOccurred())
		})

		it("creates an online cnb archive", func() {
			dep, err := metadata.UpdateDependency(metadata.Dependency{
				Name:    "some-cnb",
				Version: "1.0.0",
				URI:     "https://example.com/cnb.tgz",
				SHA256:  "some-sha",
			}, depPath)
			Expect(err).NotTo(HaveOccurred())

			Expect(dep.URI).To(Equal("file://" + depPath))
			Expect(dep.SHA256).To(Equal("84efae3d2c9ebecf21fe40ad397ba46ec8b0cc71155ac309f72e03f1347bc8e8"))
		})
	})

	when("BuildpackToml", func() {
		when("given a valid buildpack.toml", func() {
			it("parses successfully", func() {
				obp := metadata.BuildpackToml{}
				Expect(obp.Load(filepath.Join("testdata", "valid-obp.toml"))).To(Succeed())
			})
		})
	})
}
