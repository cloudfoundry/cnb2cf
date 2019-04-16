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

func TestIntegrationCreator(t *testing.T) {
	spec.Run(t, "IntegrationCreator", testIntegrationCreator, spec.Report(report.Terminal{}))
}

func testIntegrationCreator(t *testing.T, when spec.G, it spec.S) {
	var (
		configFile string
	)

	it.Before(func() {
		RegisterTestingT(t)
	})

	it("exits with an error with bad config file", func() {
		configFile = filepath.Join("testdata", "config", "bad-shim.yml")
		output, err := runCNB2CF("", "create", "-config", configFile)
		Expect(err).To(HaveOccurred())
		Expect(string(output)).To(ContainSubstring("config error"))
	})

	when("successfully running the cli", func() {
		var (
			bpName        string
			app           *cutlass.App
			shimmedBPFile string
		)

		it.Before(func() {
			configFile = filepath.Join("testdata", "config", "shim.yml")
			bpName = "shimmed_python_" + cutlass.RandStringRunes(5)

			shimmedBPFile = "python_buildpack-cflinuxfs3-1.0.0.zip"

			app = cutlass.New(filepath.Join("testdata", "python_app"))
			app.Buildpacks = []string{bpName + "_buildpack"}
		})

		it.After(func() {
			app.Destroy()
			cutlass.DeleteBuildpack(bpName)
			os.Remove(shimmedBPFile)
		})

		it("creates a runnable v2 shimmed buildpack", func() {
			output, err := runCNB2CF("", "create", "-config", configFile)
			Expect(err).NotTo(HaveOccurred(), string(output))
			Expect(cutlass.CreateOrUpdateBuildpack(bpName, shimmedBPFile, "cflinuxfs3")).To(Succeed())

			Expect(app.Push()).To(Succeed())
			Eventually(func() ([]string, error) { return app.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))
			Expect(app.GetBody("/")).To(Equal("Hello, World!"))
		})
	})
}
