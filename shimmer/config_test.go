package shimmer_test

import (
	"testing"

	"github.com/cloudfoundry/cnb2cf/shimmer"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestConfigUnit(t *testing.T) {
	spec.Run(t, "ConfigUnit", testConfigUnit, spec.Report(report.Terminal{}))
}

func testConfigUnit(t *testing.T, when spec.G, it spec.S) {
	var (
		err error
	)
	it.Before(func() {
		RegisterTestingT(t)

		Expect(err).ToNot(HaveOccurred())
		RegisterTestingT(t)
	})

	when("ValidateConfig", func() {
		it("passes with good config", func() {
			cfg := shimmer.Config{
				Language: "some-language",
				Version:  "some-version",
				Stack:    "some-stack",
				Buildpacks: []shimmer.V2Dependency{{
					Name: "some-cnb",
				}},
				Groups: []shimmer.CNBGroup{{
					Buildpacks: []shimmer.CNBBuildpack{{
						ID: "some-cnb",
					}},
				}},
			}
			Expect(shimmer.ValidateConfig(cfg)).To(Succeed())
		})
		it("errors when the buildpack IDs don't match", func() {
			cfg := shimmer.Config{
				Language: "some-language",
				Version:  "some-version",
				Stack:    "some-stack",
				Buildpacks: []shimmer.V2Dependency{{
					Name: "some-cnb",
				}},
				Groups: []shimmer.CNBGroup{{
					Buildpacks: []shimmer.CNBBuildpack{{
						ID: "some-OTHER-cnb",
					}},
				}},
			}
			Expect(shimmer.ValidateConfig(cfg).Error()).To(ContainSubstring("buildpack name some-cnb does not exist in any groups"))
		})
	})
}
