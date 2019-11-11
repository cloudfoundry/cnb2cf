package cloudnative_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/cnb2cf/cloudnative"

	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testManifest(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect func(interface{}, ...interface{}) Assertion

		tmpDir string
	)

	it.Before(func() {
		Expect = NewWithT(t).Expect

		var err error
		tmpDir, err = ioutil.TempDir("", "manifest")
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	when("UpdatingStacks", func() {
		it("should transform a v3 stackname to a v2 stackname", func() {
			v3Stacks := []string{"org.cloudfoundry.stacks.cflinuxfs3", "otherorg.stacks.bionic"}

			transformedStacks := cloudnative.UpdateStacks(v3Stacks)
			Expect(transformedStacks).To(Equal(
				[]string{"cflinuxfs3", "bionic"},
			))
		})
	})

	when("NewManifest", func() {
		it("returns a new manifest with the given buildpack details and dependencies", func() {
			manifest := cloudnative.NewManifest("org.cloudfoundry.some-language", []cloudnative.BuildpackMetadataDependency{
				{
					ID:           "some-dependency",
					Version:      "some-dependency-version",
					URI:          "some-dependency-uri",
					SHA256:       "some-dependency-sha256",
					Source:       "some-dependency-source",
					SourceSHA256: "some-dependency-source-sha256",
					Stacks:       []string{"org.cloudfoundry.stacks.cflinuxfs3", "otherorg.stacks.bionic"},
				},
			})
			Expect(manifest).To(Equal(cloudnative.Manifest{
				Language: "some-language",
				IncludeFiles: []string{
					"bin/compile",
					"bin/detect",
					"bin/finalize",
					"bin/release",
					"bin/supply",
					"buildpack.toml",
					"manifest.yml",
					"VERSION",
				},
				Dependencies: []cloudnative.ManifestDependency{
					{
						Name:         "some-dependency",
						ID:           "some-dependency",
						SHA256:       "some-dependency-sha256",
						Stacks:       []string{"cflinuxfs3", "bionic"},
						URI:          "some-dependency-uri",
						Version:      "some-dependency-version",
						Source:       "some-dependency-source",
						SourceSHA256: "some-dependency-source-sha256",
					},
				},
			}))
		})
	})

	when("WriteManifest", func() {
		it("writes a manifest to a file", func() {
			path := filepath.Join(tmpDir, "manifest.yml")

			err := cloudnative.WriteManifest(cloudnative.Manifest{
				Language:     "some-language",
				IncludeFiles: []string{"some-file"},
				Dependencies: []cloudnative.ManifestDependency{
					{
						Name:         "some-dependency",
						ID:           "some-dependency",
						SHA256:       "some-dependency-sha256",
						Stacks:       []string{"cflinuxfs3", "bionic"},
						URI:          "some-dependency-uri",
						Version:      "some-dependency-version",
						Source:       "some-dependency-source",
						SourceSHA256: "some-dependency-source-sha256",
					},
				},
			}, path)
			Expect(err).NotTo(HaveOccurred())

			contents, err := ioutil.ReadFile(path)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(contents)).To(MatchYAML(`---
language: some-language
include_files:
- some-file
dependencies:
- name: some-dependency
  id: some-dependency
  uri: some-dependency-uri
  sha256: some-dependency-sha256
  source: some-dependency-source
  source_sha256: some-dependency-source-sha256
  version: some-dependency-version
  cf_stacks:
  - cflinuxfs3
  - bionic
`))
		})

		when("failure cases", func() {
			when("a file at the given path cannot be created", func() {
				it("returns an error", func() {
					err := cloudnative.WriteManifest(cloudnative.Manifest{}, "some/made/up/path.yml")
					Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
				})
			})
		})
	})
}
