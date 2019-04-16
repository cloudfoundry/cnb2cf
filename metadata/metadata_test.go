package metadata_test

import (
	"github.com/cloudfoundry/cnb2cf/metadata"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

func TestMetadata(t *testing.T) {
	spec.Run(t, "UnitMetadata", testUnitMetadata, spec.Report(report.Terminal{}))
}

func testUnitMetadata(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("ManifestYAML", func() {
		it("loads", func() {
			m := metadata.ManifestYAML{}
			Expect(m.Load(filepath.Join("testdata", "fake-manifest.yml"))).To(Succeed())
			dep := m.Dependencies[0]
			Expect(dep.Name).To(Equal("org.cloudfoundry.buildpacks.some-language"))
			Expect(dep.URI).To(Equal("some-uri"))
			Expect(dep.SHA256).To(Equal("some-sha"))
			Expect(dep.Source).To(Equal("some-source"))
			Expect(dep.SourceSHA256).To(Equal("some-source-sha"))
		})
	})
}
