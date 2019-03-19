package integration_test

import (
	"github.com/cloudfoundry/libbuildpack/cutlass"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestIntegration(t *testing.T) {
	spec.Run(t, "Integration", testIntegration, spec.Report(report.Terminal{}))
}

func runCLI(args ...string) (string, error) {
	binary := filepath.Join("..", "build", "cnb2cf")
	cmd := exec.Command(binary, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func testIntegration(t *testing.T, when spec.G, it spec.S) {
	var (
		configFile string
	)

	it.Before(func() {
		RegisterTestingT(t)
	})

	it("exits with an error if wrong number of args given", func() {
		output, err := runCLI()
		Expect(err).To(HaveOccurred())
		Expect(string(output)).To(ContainSubstring("Wrong number of arguments, expected 1 got 0"))
	})

	it("exits with an error with bad config file", func() {
		configFile = filepath.Join("testdata", "config", "bad-shim.yml")
		output, err := runCLI(configFile)
		Expect(err).To(HaveOccurred())
		Expect(string(output)).To(ContainSubstring("config error"))
	})

	when("successfully running the cli", func() {
		var (
			bpName string
			app *cutlass.App
			shimmedBPFile string
		)

		it.Before(func() {
			configFile = filepath.Join("testdata", "config", "shim.yml")
			bpName = "shimmed_python_" + cutlass.RandStringRunes(5)

			shimmedBPFile = "python_buildpack-cflinuxfs3-1.0.0.zip"

			app = cutlass.New(filepath.Join("testdata", "python_app"))
			app.Buildpacks = []string{bpName+"_buildpack"}
		})

		it.After(func(){
			app.Destroy()
			cutlass.DeleteBuildpack(bpName)
			os.Remove(shimmedBPFile)
		})

		it("creates a runnable v2 shimmed buildpack", func() {
			output, err := runCLI(configFile)
			Expect(err).NotTo(HaveOccurred(), string(output))
			Expect(cutlass.CreateOrUpdateBuildpack(bpName, shimmedBPFile, "cflinuxfs3")).To(Succeed())

			Expect(app.Push()).To(Succeed())
			Eventually(func() ([]string, error) { return app.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))
			Expect(app.GetBody("/")).To(Equal("Hello, World!"))
		})
	})
}
