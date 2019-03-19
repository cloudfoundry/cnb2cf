package shimmer_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/cloudfoundry/cnb2cf/shimmer"
	"gopkg.in/yaml.v2"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestShimmerUnit(t *testing.T) {
	spec.Run(t, "ShimmerUnit", testShimmerUnit, spec.Report(report.Terminal{}))
}

func testShimmerUnit(t *testing.T, when spec.G, it spec.S) {
	var (
		cfg     shimmer.Config
		tempDir string
		err     error
	)
	it.Before(func() {
		RegisterTestingT(t)
		tempDir, err = ioutil.TempDir("", "")

		Expect(err).ToNot(HaveOccurred())
		RegisterTestingT(t)
	})

	it.After(func() {
		Expect(os.RemoveAll(tempDir)).To(Succeed())
	})

	when("with valid args", func() {
		it.Before(func() {
			cfg = shimmer.Config{
				Language: "some-language",
				Version:  "some-v2-version",
				Stack:    "some-stack",
				Buildpacks: []shimmer.V2Dependency{{
					Name:     "some-cnb-id",
					URI:      "some-uri",
					SHA256:   "some-sha",
					Version:  "some-version",
					CFStacks: []string{"some-stack"},
				}},
				Groups: []shimmer.CNBGroup{{
					Buildpacks: []shimmer.CNBBuildpack{{
						ID: "some-cnb-id",
					}},
				}},
			}

			Expect(shimmer.CreateBuildpack(cfg, tempDir)).To(Succeed())
		})
		it("copies over files from the v2 shim template", func() {
			Expect(filepath.Join(tempDir, "bin")).To(BeADirectory())
			Expect(filepath.Join(tempDir, "manifest.yml")).To(BeAnExistingFile())
			Expect(filepath.Join(tempDir, "order.toml")).To(BeAnExistingFile())
			Expect(filepath.Join(tempDir, "VERSION")).To(BeAnExistingFile())
		})

		it("creates a VERSION file", func() {
			contents, err := ioutil.ReadFile(filepath.Join(tempDir, "VERSION"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(Equal("some-v2-version"))
		})

		it("creates a manifest.yml", func() {
			contents, err := ioutil.ReadFile(filepath.Join(tempDir, "manifest.yml"))
			Expect(err).NotTo(HaveOccurred())
			manifest := shimmer.ManifestYAML{}
			Expect(yaml.Unmarshal(contents, &manifest)).To(Succeed())
			Expect(manifest.Language).To(Equal("some-language"))
			Expect(manifest.Stack).To(Equal("some-stack"))
			Expect(manifest.IncludeFiles).To(ContainElement("bin/compile"))
			Expect(manifest.IncludeFiles).To(ContainElement("bin/detect"))
			Expect(manifest.IncludeFiles).To(ContainElement("bin/finalize"))
			Expect(manifest.IncludeFiles).To(ContainElement("bin/release"))
			Expect(manifest.IncludeFiles).To(ContainElement("bin/supply"))
			Expect(manifest.IncludeFiles).To(ContainElement("order.toml"))
			Expect(manifest.IncludeFiles).To(ContainElement("manifest.yml"))
			Expect(manifest.IncludeFiles).To(ContainElement("VERSION"))

			Expect(manifest.Dependencies[0].Name).To(Equal("v3-builder"))
			Expect(manifest.Dependencies[1].Name).To(Equal("v3-detector"))
			Expect(manifest.Dependencies[2].Name).To(Equal("v3-launcher"))

			Expect(manifest.Dependencies[3].Name).To(Equal("some-cnb-id"))
			Expect(manifest.Dependencies[3].Version).To(Equal("some-version"))
			Expect(manifest.Dependencies[3].SHA256).To(Equal("some-sha"))
			Expect(manifest.Dependencies[3].URI).To(Equal("some-uri"))
			Expect(manifest.Dependencies[3].CFStacks).To(Equal([]string{"some-stack"}))
		})

		it("generates an order.toml", func() {
			contents, err := ioutil.ReadFile(filepath.Join(tempDir, "order.toml"))
			Expect(err).NotTo(HaveOccurred())
			orderTOML := shimmer.OrderTOML{}
			Expect(toml.Unmarshal(contents, &orderTOML)).To(Succeed())
			Expect(orderTOML.Groups[0].Buildpacks).To(Equal([]shimmer.CNBBuildpack{{ID: "some-cnb-id", Version: "latest"}}))
		})
	}, spec.Random())
}
