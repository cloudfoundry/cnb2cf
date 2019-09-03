package integration_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudfoundry/libbuildpack/cutlass"
	"github.com/cloudfoundry/dagger"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

var suite = spec.New("Integration", spec.Parallel(), spec.Report(report.Terminal{}))

func init() {
	suite("IntegrationPackager", testIntegrationPackager)
}

func TestIntegrationPackager(t *testing.T) {
	dagger.SyncParallelOutput(func() {
		suite.Run(t)
	})
}

func testIntegrationPackager(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect     func(interface{}, ...interface{}) Assertion
		Eventually func(interface{}, ...interface{}) AsyncAssertion
	)

	it.Before(func() {
		g := NewWithT(t)
		Expect = g.Expect
		Eventually = g.Eventually
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

			app = cutlass.New(filepath.Join("testdata", "nodejs_app"))
			app.Buildpacks = []string{bpName + "_buildpack"}
		})

		it.After(func() {
			Expect(app.Destroy()).To(Succeed())
			Expect(cutlass.DeleteBuildpack(bpName)).To(Succeed())
			Expect(os.Remove(shimmedBPFile)).To(Succeed())
		})

		it("creates a runnable online v2 shimmed buildpack", func() {
			output, err := runCNB2CF(bpDir, "package", "-stack", "cflinuxfs3")
			Expect(err).NotTo(HaveOccurred(), string(output))

			shimmedBPFile = filepath.Join(bpDir, "nodejs_buildpack-cflinuxfs3-v1.0.0.zip")
			Expect(cutlass.CreateOrUpdateBuildpack(bpName, shimmedBPFile, "cflinuxfs3")).To(Succeed())

			Expect(app.Push()).To(Succeed())
			Eventually(func() ([]string, error) { return app.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))
			Eventually(app.Stdout.ANSIStrippedString).Should(MatchRegexp(`Downloading from .*node`))
			Expect(app.GetBody("/")).To(Equal("Hello World!"))
		})

		it("creates a runnable offline v2 shimmed buildpack", func() {
			output, err := runCNB2CF(bpDir, "package", "-stack", "cflinuxfs3", "-cached")
			Expect(err).NotTo(HaveOccurred(), string(output))

			shimmedBPFile = filepath.Join(bpDir, "nodejs_buildpack-cached-cflinuxfs3-v1.0.0.zip")
			Expect(cutlass.CreateOrUpdateBuildpack(bpName, shimmedBPFile, "cflinuxfs3")).To(Succeed())

			Expect(app.Push()).To(Succeed())
			Eventually(func() ([]string, error) { return app.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))
			Eventually(app.Stdout.ANSIStrippedString).Should(MatchRegexp(`Node Engine.*Contributing to layer\n.*Reusing cached download from buildpack`))
			Expect(app.GetBody("/")).To(Equal("Hello World!"))
		})

		it("creates a runnable online v2 shimmed buildpack with local sources", func() {
			bpDir, err = filepath.Abs(filepath.Join("testdata", "shimmed_buildpack_with_local_sources"))
			Expect(err).NotTo(HaveOccurred())

			output, err := runCNB2CF(bpDir, "package", "-stack", "cflinuxfs3", "-dev")
			Expect(err).NotTo(HaveOccurred(), string(output))

			shimmedBPFile = filepath.Join(bpDir, "nodejs_buildpack-cflinuxfs3-v1.0.0.zip")
			Expect(cutlass.CreateOrUpdateBuildpack(bpName, shimmedBPFile, "cflinuxfs3")).To(Succeed())

			Expect(app.Push()).To(Succeed())
			Eventually(func() ([]string, error) { return app.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))
			Eventually(app.Stdout.ANSIStrippedString).Should(MatchRegexp(`Downloading from .*node`))
			Expect(app.GetBody("/")).To(Equal("Hello World!"))
		})
	})
}
