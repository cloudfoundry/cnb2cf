package creator_test

import (
	"testing"

	"github.com/cloudfoundry/cnb2cf/metadata"

	"github.com/cloudfoundry/cnb2cf/creator"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestConfigUnit(t *testing.T) {
	spec.Run(t, "ConfigUnit", testConfigUnit, spec.Report(report.Terminal{}))
}

func testConfigUnit(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect func(interface{}, ...interface{}) Assertion
	)

	it.Before(func() {
		Expect = NewWithT(t).Expect
	})

	when("ValidateConfig", func() {
		it("passes with good config", func() {
			cfg := creator.Config{
				Language: "some-language",
				Version:  "some-version",
				Stack:    "some-stack",
				Buildpacks: []metadata.V2Dependency{{
					Name: "some-cnb",
				}},
				Groups: []metadata.CNBGroup{{
					Buildpacks: []metadata.CNBBuildpack{{
						ID: "some-cnb",
					}},
				}},
			}
			Expect(creator.ValidateConfig(cfg)).To(Succeed())
		})
		it("errors when the buildpack IDs don't match", func() {
			cfg := creator.Config{
				Language: "some-language",
				Version:  "some-version",
				Stack:    "some-stack",
				Buildpacks: []metadata.V2Dependency{{
					Name: "some-cnb",
				}},
				Groups: []metadata.CNBGroup{{
					Buildpacks: []metadata.CNBBuildpack{{
						ID: "some-OTHER-cnb",
					}},
				}},
			}
			Expect(creator.ValidateConfig(cfg).Error()).To(ContainSubstring("buildpack name some-cnb does not exist in any groups"))
		})
	})
}
