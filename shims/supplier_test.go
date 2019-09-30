package shims_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudfoundry/cnb2cf/shims"
	"github.com/cloudfoundry/libbuildpack"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testSupplier(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect func(interface{}, ...interface{}) Assertion

		supplier        shims.Supplier
		v2AppDir        string
		v2BuildpacksDir string
		v3AppDir        string
		v3BuildpacksDir string
		v2DepsDir       string
		v2CacheDir      string
		depsIndex       string
		tempDir         string
		orderDir        string
		installer       *shims.CNBInstaller
		manifest        *libbuildpack.Manifest
		buffer          *bytes.Buffer
		logger          *libbuildpack.Logger
	)

	it.Before(func() {
		Expect = NewWithT(t).Expect

		var err error

		tempDir, err = ioutil.TempDir("", "tmp")
		Expect(err).NotTo(HaveOccurred())

		v2AppDir = filepath.Join(tempDir, "app")
		Expect(os.MkdirAll(v2AppDir, 0777)).To(Succeed())

		v3AppDir = filepath.Join(tempDir, "cnb-app")

		v2DepsDir = filepath.Join(tempDir, "deps")
		depsIndex = "0"
		Expect(os.MkdirAll(filepath.Join(v2DepsDir, depsIndex), 0777)).To(Succeed())

		v2CacheDir = filepath.Join(tempDir, "cache")
		Expect(os.MkdirAll(v2CacheDir, 0777)).To(Succeed())

		buffer = new(bytes.Buffer)
		logger = libbuildpack.NewLogger(buffer)

		manifest, err = libbuildpack.NewManifest(filepath.Join("testdata", "buildpack"), logger, time.Now())
		Expect(err).ToNot(HaveOccurred())
		installer = shims.NewCNBInstaller(manifest)

		orderDir = filepath.Join(tempDir, "order")
		Expect(os.MkdirAll(orderDir, 0777)).To(Succeed())

		v3BuildpacksDir = filepath.Join(tempDir, "v3Buildpacks")
		v2BuildpacksDir = filepath.Join(tempDir, "v2Buildpacks")

		Expect(os.MkdirAll(filepath.Join(v2DepsDir, depsIndex), 0777)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(v2BuildpacksDir, depsIndex), 0777)).To(Succeed())

		supplier = shims.Supplier{
			V2AppDir:        v2AppDir,
			V2BuildpackDir:  filepath.Join(v2BuildpacksDir, depsIndex),
			V3AppDir:        v3AppDir,
			V2DepsDir:       v2DepsDir,
			V2CacheDir:      v2CacheDir,
			DepsIndex:       depsIndex,
			V3BuildpacksDir: v3BuildpacksDir,
			OrderDir:        orderDir,
			Installer:       installer,
			Manifest:        manifest,
			Logger:          logger,
		}
	})

	it.After(func() {
		Expect(os.RemoveAll(tempDir)).To(Succeed())
	})

	when("SetUpFirstV3Buildpack", func() {
		it("Moves V2AppDir to V3AppDir if it has not already been moved", func() {
			Expect(v3AppDir).NotTo(BeADirectory())
			Expect(supplier.SetUpFirstV3Buildpack()).To(Succeed())
			Expect(v3AppDir).To(BeADirectory())
		})

		it("Writes a sentinel file in the app dir", func() {
			Expect(supplier.SetUpFirstV3Buildpack()).To(Succeed())
			Expect(filepath.Join(v3AppDir, ".cloudfoundry", libbuildpack.SENTINEL)).To(BeAnExistingFile())
		})

		it("Writes a symlink at the V2AppDir to a fake file with a clear error message", func() {
			Expect(supplier.SetUpFirstV3Buildpack()).To(Succeed())

			linkDst, err := os.Readlink(v2AppDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(linkDst).To(Equal(shims.ERROR_FILE))
		})

		it("Does nothing if V2AppDir has already been moved", func() {
			Expect(os.Remove(v2AppDir)).To(Succeed())
			Expect(os.Symlink("some-file", v2AppDir)).To(Succeed())
			Expect(supplier.SetUpFirstV3Buildpack()).To(Succeed())
		})
	})

	when("RemoveV2DepsIndex", func() {
		it("removes the V2 deps index so no one writes to it", func() {
			Expect(supplier.RemoveV2DepsIndex()).To(Succeed())
			Expect(filepath.Join(v2DepsDir, depsIndex)).ToNot(BeAnExistingFile())
		})
	})

	when("SaveBuildpackToml", func() {
		it("copies the order metadata to be used for finalize", func() {
			Expect(ioutil.WriteFile(filepath.Join(v2BuildpacksDir, depsIndex, "buildpack.toml"), []byte(""), 0666)).To(Succeed())

			orderFile, err := supplier.SaveBuildpackToml()
			Expect(err).NotTo(HaveOccurred())
			Expect(orderFile).To(Equal(filepath.Join(orderDir, "buildpack"+depsIndex+".toml")))
			Expect(orderFile).To(BeAnExistingFile())
		})
	})

	when("CheckBuildpackValid", func() {
		it.Before(func() {
			Expect(os.Setenv("CF_STACK", "cflinuxfs2")).To(Succeed())
		})

		when("buildpack is valid", func() {
			it("it logs the buildpack name and version", func() {
				Expect(supplier.CheckBuildpackValid()).To(Succeed())

				Expect(buffer.String()).To(ContainSubstring("-----> SomeName Buildpack version 0.0.1"))
			})
		})
	})
}
