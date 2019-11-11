package commands_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/cnb2cf/cloudnative"
	"github.com/cloudfoundry/cnb2cf/cloudnative/untested/fakes"
	"github.com/cloudfoundry/cnb2cf/commands"

	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
)

func TestUnitPackageCommand(t *testing.T) {
	spec.Run(t, "Node", testPackageCommand, spec.Report(report.Terminal{}))
}

func testPackageCommand(t *testing.T, when spec.G, it spec.S) {
	//var pkg commands.Package
	it.Before(func() {
		RegisterTestingT(t)

	})

	when("calling Fetch", func() {
		it("fetches dependencies in a buildpack", func() {
			mockInstaller := fakes.Installer{}
			bpTOMLStub := filepath.Join("testdata", "buildpack.toml")

			buildpack, err := cloudnative.ParseBuildpack(bpTOMLStub)
			Expect(err).NotTo(HaveOccurred())

			err = commands.Fetch(buildpack, &mockInstaller)

			Expect(err).NotTo(HaveOccurred())
			Expect(mockInstaller.DownloadCall.CallCount).To(Equal(3))
			Expect(mockInstaller.DownloadCall.Receives.Uri).To(Equal("https://github.com/cloudfoundry/nodejs-cnb/releases/download/v0.0.2/nodejs-cnb-compat-cflinuxfs3-v0.0.2.zip"))
			Expect(mockInstaller.DownloadCall.Receives.Checksum).To(Equal("a5d53002f97f380b40dace0112c21f4b2b60ba382a0fb499ca7c74c2d8fc44e9"))
			destination := filepath.Join(os.TempDir(), "downloads", "")
			Expect(mockInstaller.DownloadCall.Receives.Destination).To(Equal(destination))
		})
	})
}
