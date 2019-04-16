package integration_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestIntegrationPackager(t *testing.T) {
	spec.Run(t, "IntegrationPackager", testIntegrationPackager, spec.Report(report.Terminal{}))
}

func testIntegrationPackager(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("successfully running the packaging command", func() {
		var (
			bpName, bpDir, shimmedBPFile string
			app                          *cutlass.App
			err                          error
		)

		it.Before(func() {
			bpName = "shimmed_nodejs_" + cutlass.RandStringRunes(5)
			bpDir, err = filepath.Abs(filepath.Join("testdata", "shimmed_buildpack"))
			Expect(err).NotTo(HaveOccurred())
			shimmedBPFile = filepath.Join(bpDir, "nodejs_buildpack-cached-cflinuxfs3-v1.0.0.zip")

			app = cutlass.New(filepath.Join("testdata", "nodejs_app"))
			app.Buildpacks = []string{bpName + "_buildpack"}
		})

		it.After(func() {
			app.Destroy()
			cutlass.DeleteBuildpack(bpName)
			os.Remove(shimmedBPFile)
		})

		it("creates a runnable online v2 shimmed buildpack", func() {
			output, err := runCNB2CF(bpDir, "package", "-stack", "cflinuxfs3")
			Expect(err).NotTo(HaveOccurred(), string(output))
			Expect(cutlass.CreateOrUpdateBuildpack(bpName, shimmedBPFile, "cflinuxfs3")).To(Succeed())

			Expect(app.Push()).To(Succeed())
			Eventually(func() ([]string, error) { return app.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))
			Eventually(app.Stdout.ANSIStrippedString).Should(MatchRegexp(`Downloading from .*node`))
			Expect(app.GetBody("/")).To(Equal("Hello World!"))
		})

		it("creates a runnable offline v2 shimmed buildpack", func() {
			output, err := runCNB2CF(bpDir, "package", "-stack", "cflinuxfs3", "-cached")
			Expect(err).NotTo(HaveOccurred(), string(output))
			Expect(cutlass.CreateOrUpdateBuildpack(bpName, shimmedBPFile, "cflinuxfs3")).To(Succeed())

			Expect(app.Push()).To(Succeed())
			Eventually(func() ([]string, error) { return app.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))
			Eventually(app.Stdout.ANSIStrippedString).Should(MatchRegexp(`NodeJS.*Contributing to layer\n.*Reusing cached download from buildpack`))
			Expect(app.GetBody("/")).To(Equal("Hello World!"))
		})
	})
}
