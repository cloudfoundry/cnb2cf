package integration_test

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudfoundry/cnb2cf/utils"
	"github.com/cloudfoundry/dagger"
	"github.com/cloudfoundry/libbuildpack/cutlass"
	"github.com/onsi/ginkgo"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

var suite = spec.New("Integration", spec.Sequential(), spec.Report(report.Terminal{}))

func init() {
	suite("Integration", testIntegration)
}

func TestIntegration(t *testing.T) {
	dagger.SyncParallelOutput(func() {
		suite.Run(t)
	})
}

func testIntegration(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect     func(interface{}, ...interface{}) Assertion
		Eventually func(interface{}, ...interface{}) AsyncAssertion
	)

	it.Before(func() {
		g := NewWithT(t)
		Expect = g.Expect
		Eventually = g.Eventually
		cutlass.DefaultStdoutStderr = ginkgo.GinkgoWriter
	})

	when("successfully running the packaging command", func() {
		var (
			bpName, bpDir, shimmedBPFile, testDir string
			app                                   *cutlass.App
		)

		it.Before(func() {
			var err error
			testDir, err = ioutil.TempDir("", "integration")
			Expect(err).NotTo(HaveOccurred())

			bpDir, err = filepath.Abs(filepath.Join("testdata", "metabuildpack"))
			Expect(err).NotTo(HaveOccurred())

			original, err := os.Open(filepath.Join(bpDir, "buildpack.toml"))
			Expect(err).NotTo(HaveOccurred())
			defer original.Close()

			duplicate, err := os.Create(filepath.Join(testDir, "buildpack.toml"))
			Expect(err).NotTo(HaveOccurred())
			defer duplicate.Close()

			_, err = io.Copy(duplicate, original)
			Expect(err).NotTo(HaveOccurred())

			bpName = "nodejs"
			bpDir = testDir

			app = cutlass.New(filepath.Join("testdata", "nodejs_app"))
			app.Buildpacks = []string{bpName + "_buildpack"}
		})

		it.After(func() {
			Expect(app.Destroy()).To(Succeed())
			Expect(cutlass.DeleteBuildpack(bpName)).To(Succeed())
			Expect(os.Remove(shimmedBPFile)).To(Succeed())
			Expect(os.RemoveAll(testDir)).To(Succeed())
		})

		it("creates a runnable online v2 shimmed buildpack", func() {
			output, err := runCNB2CF(bpDir, "package", "-stack", "cflinuxfs3")
			Expect(err).NotTo(HaveOccurred(), string(output))

			shimmedBPFile = filepath.Join(bpDir, "cf-nodejs_buildpack-cflinuxfs3-v1.0.0.zip")

				desiredBPFiles := []string{
					"buildpack.toml",
					"manifest.yml",
					"VERSION",
					"bin/compile",
					"bin/detect",
					"bin/finalize",
					"bin/release",
					"bin/supply",
				}

				presentBPFiles, err := utils.GetFilesFromZip(shimmedBPFile)
				Expect(err).ToNot(HaveOccurred())
				for _, file := range desiredBPFiles {
					Expect(presentBPFiles).To(ContainElement(file))
				}

				fileContents, err := utils.GetFileContentsFromZip(shimmedBPFile, "manifest.yml")
				Expect(err).ToNot(HaveOccurred())
				Expect(string(fileContents)).To(ContainSubstring("lifecycle"))

				Expect(cutlass.CreateOrUpdateBuildpack(bpName, shimmedBPFile, "cflinuxfs3")).To(Succeed())

				Expect(app.Push()).To(Succeed())
				Eventually(func() ([]string, error) { return app.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))
				Eventually(app.Stdout.ANSIStrippedString).Should(MatchRegexp(`Downloading from .*node`))
				Expect(app.GetBody("/")).To(Equal("Hello World!"))
			})

			// create a new nodejs buildpack
			it("creates a runnable offline v2 shimmed buildpack", func() {
				output, err := runCNB2CF(bpDir, "package", "-stack", "cflinuxfs3", "-cached")
				Expect(err).NotTo(HaveOccurred(), string(output))
				app.Buildpacks = []string{bpName+"_buildpack"}
				shimmedBPFile = filepath.Join(bpDir, "cf-nodejs_buildpack-cached-cflinuxfs3-v1.0.0.zip")
				Expect(cutlass.CreateOrUpdateBuildpack(bpName, shimmedBPFile, "cflinuxfs3")).To(Succeed())

				Expect(app.Push()).To(Succeed())
				Eventually(func() ([]string, error) { return app.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))
				Eventually(app.Stdout.ANSIStrippedString).Should(MatchRegexp(`Node Engine.*Contributing to layer\n.*Reusing cached download from buildpack`))
				Expect(app.GetBody("/")).To(Equal("Hello World!"))
			})

			// TODO: needs to wait for the buildpack.toml from the nodejs-cnb to contain the sources
			it.Pend("creates a runnable online v2 shimmed buildpack with local sources", func() {
				Expect(cutlass.CreateOrUpdateBuildpack(bpName, shimmedBPFile, "cflinuxfs3")).To(Succeed())

				Expect(app.Push()).To(Succeed())
				Eventually(func() ([]string, error) { return app.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))
				Eventually(app.Stdout.ANSIStrippedString).Should(MatchRegexp(`Downloading from .*node`))
				Expect(app.GetBody("/")).To(Equal("Hello World!"))
			})

			when("the buildpack is not specified during push", func() {
				it("creates a runnable online v2 shimmed buildpack", func() {
					output, err := runCNB2CF(bpDir, "package", "-stack", "cflinuxfs3")
					Expect(err).NotTo(HaveOccurred(), string(output))

					shimmedBPFile = filepath.Join(bpDir, "cf-nodejs_buildpack-cflinuxfs3-v1.0.0.zip")

					desiredBPFiles := []string{
						"buildpack.toml",
						"manifest.yml",
						"VERSION",
						"bin/compile",
						"bin/detect",
						"bin/finalize",
						"bin/release",
						"bin/supply",
					}

					presentBPFiles, err := utils.GetFilesFromZip(shimmedBPFile)
					Expect(err).ToNot(HaveOccurred())
					for _, file := range desiredBPFiles {
						Expect(presentBPFiles).To(ContainElement(file))
					}

					fileContents, err := utils.GetFileContentsFromZip(shimmedBPFile, "manifest.yml")
					Expect(err).ToNot(HaveOccurred())
					Expect(string(fileContents)).To(ContainSubstring("lifecycle"))

					Expect(cutlass.CreateOrUpdateBuildpack(bpName, shimmedBPFile, "cflinuxfs3")).To(Succeed())

					Expect(app.Push()).To(Succeed())
					Eventually(func() ([]string, error) { return app.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))
					Eventually(app.Stdout.ANSIStrippedString).Should(MatchRegexp(`Downloading from .*node`))
					Expect(app.GetBody("/")).To(Equal("Hello World!"))
				})
		})
	})

}
