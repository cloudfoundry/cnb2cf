package integration_test

import (
	"github.com/onsi/ginkgo"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
)

func init() {
	suite("IntegrationCreator", testIntegrationCreator)
}

func testIntegrationCreator(t *testing.T, when spec.G, it spec.S) {
	var (
		configFile string
		Expect     func(interface{}, ...interface{}) Assertion
		Eventually func(interface{}, ...interface{}) AsyncAssertion
	)

	it.Before(func() {
		g := NewWithT(t)
		Expect = g.Expect
		Eventually = g.Eventually
		cutlass.DefaultStdoutStderr = ginkgo.GinkgoWriter
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
			Expect(app.Destroy()).To(Succeed())
			Expect(cutlass.DeleteBuildpack(bpName)).To(Succeed())
			Expect(os.Remove(shimmedBPFile)).To(Succeed())
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
