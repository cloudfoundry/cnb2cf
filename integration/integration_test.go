package integration_test

import (
	"github.com/cloudfoundry/libbuildpack"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestIntegration(t *testing.T) {
	spec.Run(t, "Integration", testIntegration, spec.Report(report.Terminal{}))
}

func runCLI(location string, args ...string) (string, error) {
	rootDir, err := cutlass.FindRoot()
	if err != nil {
		return "", err
	}

	if location != "" { // location => /tmp/location
		binary := filepath.Join(rootDir, "build", "cnb2cf") // run /tmp/location/build/cnb2cf (we WANT /Users/pivotal/workspace/cnb2cf/build/cnb2cf)
		cmd := exec.Command(binary, args...)
		cmd.Dir = location
		output, err := cmd.CombinedOutput()
		return string(output), err
	} else{
		binary := filepath.Join(rootDir, "build", "cnb2cf")
		cmd := exec.Command(binary, args...)
		cmd.Dir = rootDir
		output, err := cmd.CombinedOutput()
		return string(output), err
	}

}

func testIntegration(t *testing.T, when spec.G, it spec.S) {
	var (
		configFile string
	)

	it.Before(func() {
		RegisterTestingT(t)
	})

	it("exits with an error if wrong number of args given", func() {
		output, err := runCLI("", "some", "invalid", "number", "of", "args")
		Expect(err).To(HaveOccurred())
		Expect(string(output)).To(ContainSubstring("wrong number of arguments, expected 1 - 2 got 5"))
	})

	it("exits with an error with bad config file", func() {
		configFile = filepath.Join("testdata", "config", "bad-shim.yml")
		output, err := runCLI("",configFile)
		Expect(err).To(HaveOccurred())
		Expect(string(output)).To(ContainSubstring("config error"))
	})

	when("successfully running the cli with a shim.yml as the first arg", func() {
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
			output, err := runCLI("",configFile)
			Expect(err).NotTo(HaveOccurred(), string(output))
			Expect(cutlass.CreateOrUpdateBuildpack(bpName, shimmedBPFile, "cflinuxfs3")).To(Succeed())

			Expect(app.Push()).To(Succeed())
			Eventually(func() ([]string, error) { return app.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))
			Expect(app.GetBody("/")).To(Equal("Hello, World!"))
		})
	})

	when("successfully running the cli build with a valid manifest.yml", func() {
		var (
			bpName            string
			app               *cutlass.App
			location          string
			tomlFile          string
			manifestFile      string
			shimmedBPFilePath string
			err               error
		)

		it.Before(func() {
			manifestFile = filepath.Join("testdata", "config", "manifest.yml")
			tomlFile = filepath.Join("testdata", "config", "order.toml")
			shimmedBPFileName := "python_buildpack-cflinuxfs3-cached-1.0.0.zip"
			shimmedBPFilePath = filepath.Join("location", shimmedBPFileName)

			location, err = ioutil.TempDir("", "")
			Expect(err).NotTo(HaveOccurred())

			ymlFilePath := filepath.Join(location, filepath.Base(manifestFile))
			tomlFileDest := filepath.Join(location, filepath.Base(tomlFile))

			Expect(libbuildpack.CopyFile(manifestFile, ymlFilePath)).To(Succeed())
			Expect(libbuildpack.CopyFile(tomlFile, tomlFileDest)).To(Succeed())
		})

		it.After(func() {
			os.RemoveAll(location)
		})

		it.Focus("creates a runnable offline v2 shimmed buildpack", func() {
			output, err := runCLI(location, "build", "--cached")
			Expect(err).NotTo(HaveOccurred(), string(output))
			Expect(cutlass.CreateOrUpdateBuildpack(bpName, shimmedBPFilePath, "cflinuxfs3")).To(Succeed())

			Expect(app.Push()).To(Succeed())
			Eventually(func() ([]string, error) { return app.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))
			Expect(app.GetBody("/")).To(Equal("Hello, World!"))
			Expect(app.Stdout.String()).To(MatchRegexp(`Copy [.*python-cnb.*]`))
		})
	})
}
