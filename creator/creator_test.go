package creator_test

import (
	"archive/zip"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/cnb2cf/metadata"

	"github.com/BurntSushi/toml"
	"github.com/cloudfoundry/cnb2cf/creator"
	yaml "gopkg.in/yaml.v2"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestCreatorUnit(t *testing.T) {
	spec.Run(t, "CreatorUnit", testCreatorUnit, spec.Report(report.Terminal{}))
}

func testCreatorUnit(t *testing.T, when spec.G, it spec.S) {
	var (
		cfg     creator.Config
		tempDir string
		err     error
		Expect  func(interface{}, ...interface{}) Assertion
	)

	it.Before(func() {
		Expect = NewWithT(t).Expect
		tempDir, err = ioutil.TempDir("", "")

		Expect(err).ToNot(HaveOccurred())
	})

	it.After(func() {
		Expect(os.RemoveAll(tempDir)).To(Succeed())
	})

	when("CreateBuildpack", func() {
		it.Before(func() {
			cfg = creator.Config{
				Language: "some-language",
				Version:  "some-v2-version",
				Stack:    "some-stack",
				Buildpacks: []metadata.V2Dependency{{
					Name:     "some-cnb-id",
					URI:      "some-uri",
					SHA256:   "some-sha",
					Version:  "some-version",
					CFStacks: []string{"some-stack"},
				}},
				Groups: []metadata.CNBGroup{{
					Buildpacks: []metadata.CNBBuildpack{{
						ID: "some-cnb-id",
					}},
				}},
			}

			Expect(creator.CreateBuildpack(cfg, tempDir)).To(Succeed())
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
			manifest := metadata.ManifestYAML{}
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

			Expect(manifest.Dependencies[0].Name).To(Equal("lifecycle"))

			Expect(manifest.Dependencies[1].Name).To(Equal("some-cnb-id"))
			Expect(manifest.Dependencies[1].Version).To(Equal("some-version"))
			Expect(manifest.Dependencies[1].SHA256).To(Equal("some-sha"))
			Expect(manifest.Dependencies[1].URI).To(Equal("some-uri"))
			Expect(manifest.Dependencies[1].CFStacks).To(Equal([]string{"some-stack"}))
		})

		it("generates an order.toml", func() {
			contents, err := ioutil.ReadFile(filepath.Join(tempDir, "order.toml"))
			Expect(err).NotTo(HaveOccurred())
			orderTOML := metadata.OrderTOML{}
			Expect(toml.Unmarshal(contents, &orderTOML)).To(Succeed())
			Expect(orderTOML.Groups[0].Buildpacks).To(Equal([]metadata.CNBBuildpack{{ID: "some-cnb-id", Version: "latest"}}))
		})
	}, spec.Random())

	when("CreateZip", func() {
		var (
			bpDir, tempDir string
			config         creator.Config
			err            error
		)

		it.Before(func() {
			tempDir, err = ioutil.TempDir("", "")
			Expect(err).NotTo(HaveOccurred())

			config = creator.Config{
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
			Expect(creator.CreateZip(config, bpDir, tempDir)).To(Succeed())
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
