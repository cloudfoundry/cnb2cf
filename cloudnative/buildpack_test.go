package cloudnative_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/cnb2cf/cloudnative"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBuildpack(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect func(interface{}, ...interface{}) Assertion

		tmpDir string
	)

	it.Before(func() {
		Expect = NewWithT(t).Expect

		var err error
		tmpDir, err = ioutil.TempDir("", "buildpack")
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	when("Parse", func() {
		it("parses a buildpack.toml into a Buildpack", func() {
			path := filepath.Join(tmpDir, "buildpack.toml")
			contents := `
api = "0.2"

[buildpack]
id = "some-buildpack-id"
name = "some-buildpack-name"
version = "some-buildpack-version"

[metadata]
include_files = ["buildpack.toml"]

[[metadata.dependencies]]
id = "lifecycle"
sha256 = "some-lifecycle-sha256"
source = "some-lifecycle-source"
source_sha256 = "some-lifecycle-source-sha256"
stacks = ["some-stack-name"]
uri = "some-lifecycle-uri"
version = "some-lifecycle-version"

[[metadata.dependencies]]
id = "some-dependency"
sha256 = "some-dependency-sha256"
source = "some-dependency-source"
source_sha256 = "some-dependency-source-sha256"
stacks = ["some-stack-name"]
uri = "some-dependency-uri"
version = "some-dependency-version"

[[order]]

[[order.group]]
id = "some-dependency"
version = "some-dependency-version"
`

			err := ioutil.WriteFile(path, []byte(contents), 0644)
			Expect(err).NotTo(HaveOccurred())

			buildpack, err := cloudnative.ParseBuildpack(path)
			Expect(err).NotTo(HaveOccurred())

			Expect(buildpack).To(Equal(cloudnative.Buildpack{
				API: "0.2",
				Info: cloudnative.BuildpackInfo{
					ID:      "some-buildpack-id",
					Name:    "some-buildpack-name",
					Version: "some-buildpack-version",
				},
				Metadata: cloudnative.BuildpackMetadata{
					IncludeFiles: []string{"buildpack.toml"},
					Dependencies: []cloudnative.BuildpackMetadataDependency{
						{
							ID:           "lifecycle",
							SHA256:       "some-lifecycle-sha256",
							Source:       "some-lifecycle-source",
							SourceSHA256: "some-lifecycle-source-sha256",
							Stacks:       []string{"some-stack-name"},
							URI:          "some-lifecycle-uri",
							Version:      "some-lifecycle-version",
						},
						{
							ID:           "some-dependency",
							SHA256:       "some-dependency-sha256",
							Source:       "some-dependency-source",
							SourceSHA256: "some-dependency-source-sha256",
							Stacks:       []string{"some-stack-name"},
							URI:          "some-dependency-uri",
							Version:      "some-dependency-version",
						},
					},
				},
				Orders: []cloudnative.BuildpackOrder{
					{
						Groups: []cloudnative.BuildpackOrderGroup{
							{
								ID:      "some-dependency",
								Version: "some-dependency-version",
							},
						},
					},
				},
			}))
		})

		when("failure cases", func() {
			when("the file does not contain valid TOML", func() {
				it("returns an error", func() {
					path := filepath.Join(tmpDir, "buildpack.toml")
					err := ioutil.WriteFile(path, []byte("%%%"), 0644)
					Expect(err).NotTo(HaveOccurred())

					_, err = cloudnative.ParseBuildpack(path)
					Expect(err).To(MatchError(ContainSubstring(fmt.Sprintf("failed to parse %s", path))))
					Expect(err).To(MatchError(ContainSubstring("bare keys cannot contain '%'")))
				})
			})
		})
	})
}
