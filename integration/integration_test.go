package integration_test

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/shim-generator/buildpack"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestIntegration(t *testing.T) {
	spec.Run(t, "Integration", testIntegration, spec.Report(report.Terminal{}))
}

func testIntegration(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})
	when("binary created", func() {
		it("exists", func() {
			Expect("../shimmer").To(BeAnExistingFile())
		})
	})

	when("we run shimmer", func() {
		var (
			tempDir string
			err     error
		)

		it.Before(func() {
			tempDir, err = ioutil.TempDir("", "")
			Expect(err).ToNot(HaveOccurred())
		})

		it.After(func() {
			Expect(os.RemoveAll(tempDir)).To(Succeed())
		})

		it("creates a directory with the name of the first argument", func() {
			cmd := exec.Command(filepath.Join("..", "shimmer"), filepath.Join("..", "template"), filepath.Join(tempDir, "new-shim"), "testdata")
			_, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred())
			Expect(filepath.Join(tempDir, "new-shim")).To(BeADirectory())
		})

		it("exits with an error if wrong number of args given", func() {
			cmd := exec.Command(filepath.Join("..", "shimmer"), filepath.Join("..", "template"), filepath.Join(tempDir, "new-shim"))
			output, err := cmd.CombinedOutput()
			Expect(err).To(HaveOccurred())
			Expect(string(output)).To(ContainSubstring("Wrong number of args"))
		})

		//it("copies over the v2 shim template into the given directory", func() {
		//	cmd := exec.Command(filepath.Join("..", "shimmer"), filepath.Join(tempDir, "new-shim"))
		//	_, err := cmd.CombinedOutput()
		//	Expect(err).NotTo(HaveOccurred())
		//
		//	newDirChecksum := md5.Sum([]byte(filepath.Join(tempDir, "new-shim")))
		//	templateDirChecksum := md5.Sum([]byte(filepath.Join("..", "template")))
		//
		//	Expect(newDirChecksum).To(Equal(templateDirChecksum))
		//})

		it("copies over files from the v2 shim template", func() {
			cmd := exec.Command(filepath.Join("..", "shimmer"), filepath.Join("..", "template"), filepath.Join(tempDir, "new-shim"), "testdata")
			_, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred())
			Expect(filepath.Join(tempDir, "new-shim", "bin")).To(BeADirectory())
			Expect(filepath.Join(tempDir, "new-shim", "scripts")).To(BeADirectory())
			Expect(filepath.Join(tempDir, "new-shim", "manifest.yml")).To(BeAnExistingFile())
			Expect(filepath.Join(tempDir, "new-shim", "order.toml")).To(BeAnExistingFile())
		})

		it("modifies the manifest.yml with the cnb buildpack name", func() {
			cmd := exec.Command(filepath.Join("..", "shimmer"), filepath.Join("..", "template"), filepath.Join(tempDir, "new-shim"), "testdata")
			_, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred())
			contents, err := ioutil.ReadFile(filepath.Join(tempDir, "new-shim", "manifest.yml"))
			Expect(contents).NotTo(BeEmpty())
			manifest := buildpack.Manifest{}
			Expect(yaml.Unmarshal(contents, &manifest)).To(Succeed())
			deps := manifest["dependencies"].([]interface{})
			firstDep := deps[0]
			depName := firstDep.(map[interface{}]interface{})["name"]

			Expect(depName).To(Equal("test-cnb-buildpack"))
		})
	})
}
