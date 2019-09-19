package shims_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/cnb2cf/shims"
	"github.com/golang/mock/gomock"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=detector.go --destination=mocks_detector_shims_test.go --package=shims_test

func testDetector(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect func(interface{}, ...interface{}) Assertion

		detector        shims.DefaultDetector
		v3BuildpacksDir string
		v3AppDir        string
		tempDir         string
		v3LifecycleDir  string
		groupMetadata   string
		orderMetadata   string
		planMetadata    string
		mockInstaller   *MockInstaller
		mockCtrl        *gomock.Controller
	)

	it.Before(func() {
		Expect = NewWithT(t).Expect

		var err error
		mockCtrl = gomock.NewController(t)
		mockInstaller = NewMockInstaller(mockCtrl)

		tempDir, err = ioutil.TempDir("", "tmp")
		Expect(err).NotTo(HaveOccurred())

		v3AppDir = filepath.Join(tempDir, "cnb-app")

		v3LifecycleDir = filepath.Join(tempDir, "lifecycle-dir")
		Expect(os.MkdirAll(v3LifecycleDir, 0777)).To(Succeed())

		groupMetadata = filepath.Join(tempDir, "metadata", "group.toml")
		orderMetadata = filepath.Join(tempDir, "metadata", "order.toml")
		planMetadata = filepath.Join(tempDir, "metadata", "plan.toml")

		v3BuildpacksDir = filepath.Join(tempDir, "buildpacks")

		detector = shims.DefaultDetector{
			AppDir:          v3AppDir,
			V3BuildpacksDir: v3BuildpacksDir,
			V3LifecycleDir:  v3LifecycleDir,
			OrderMetadata:   orderMetadata,
			GroupMetadata:   groupMetadata,
			PlanMetadata:    planMetadata,
			Installer:       mockInstaller,
		}
	})

	it.After(func() {
		Expect(os.RemoveAll(tempDir)).To(Succeed())
	})

	it("should run the v3-detector", func() {
		mockInstaller.EXPECT().InstallCNBs(orderMetadata, v3BuildpacksDir)
		mockInstaller.EXPECT().InstallLifecycle(v3LifecycleDir).Do(func(path string) error {
			contents := "#!/usr/bin/env bash\nexit 0\n"
			return ioutil.WriteFile(filepath.Join(path, "detector"), []byte(contents), os.ModePerm)
		})
		Expect(detector.Detect()).ToNot(HaveOccurred())
	})
}
