package shims_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/cnb2cf/shims"
	"github.com/cloudfoundry/cnb2cf/shims/fakes"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testDetector(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect func(interface{}, ...interface{}) Assertion

		detector        shims.Detector
		installer       *fakes.Installer
		environment     *fakes.Environment
		fakeExecutable  *fakes.Executable
		v3BuildpacksDir string
		v3AppDir        string
		tempDir         string
		v3LifecycleDir  string
		groupMetadata   string
		orderMetadata   string
		planMetadata    string
	)

	it.Before(func() {
		Expect = NewWithT(t).Expect

		var err error
		tempDir, err = ioutil.TempDir("", "tmp")
		Expect(err).NotTo(HaveOccurred())

		v3AppDir = filepath.Join(tempDir, "cnb-app")

		v3LifecycleDir = filepath.Join(tempDir, "lifecycle-dir")
		Expect(os.MkdirAll(v3LifecycleDir, 0777)).To(Succeed())

		groupMetadata = filepath.Join(tempDir, "metadata", "group.toml")
		orderMetadata = filepath.Join(tempDir, "metadata", "order.toml")
		planMetadata = filepath.Join(tempDir, "metadata", "plan.toml")

		v3BuildpacksDir = filepath.Join(tempDir, "buildpacks")

		installer = &fakes.Installer{}
		installer.InstallLifecycleCall.Stub = func(path string) error {
			contents := "#!/usr/bin/env bash\nexit 0\n"
			return ioutil.WriteFile(filepath.Join(path, "detector"), []byte(contents), os.ModePerm)
		}

		environment = &fakes.Environment{}
		environment.ServicesCall.Returns.String = `{"some-key": "some-val"}`
		environment.StackCall.Returns.String = "some-stack"

		fakeExecutable = &fakes.Executable{}

		detector = shims.Detector{
			AppDir:          v3AppDir,
			V3BuildpacksDir: v3BuildpacksDir,
			V3LifecycleDir:  v3LifecycleDir,
			OrderMetadata:   orderMetadata,
			GroupMetadata:   groupMetadata,
			PlanMetadata:    planMetadata,
			Installer:       installer,
			Environment:     environment,
			Executor:        fakeExecutable,
		}
	})

	it.After(func() {
		Expect(os.RemoveAll(tempDir)).To(Succeed())
	})

	it("should run the v3-detector", func() {
		Expect(detector.Detect()).To(Succeed())

		Expect(installer.InstallCNBsCall.Receives.OrderFile).To(Equal(orderMetadata))
		Expect(installer.InstallCNBsCall.Receives.InstallDir).To(Equal(v3BuildpacksDir))

		Expect(installer.InstallLifecycleCall.Receives.Dst).To(Equal(v3LifecycleDir))

		Expect(environment.ServicesCall.CallCount).To(Equal(1))

		Expect(fakeExecutable.ExecuteCall.Receives.Execution.Args).To(Equal([]string{
			"-app", v3AppDir,
			"-buildpacks", v3BuildpacksDir,
			"-order", orderMetadata,
			"-group", groupMetadata,
			"-plan", planMetadata,
		}))
		Expect(fakeExecutable.ExecuteCall.Receives.Execution.Stderr).To(Equal(os.Stderr))
		Expect(fakeExecutable.ExecuteCall.Receives.Execution.Env).To(ContainElement(`CNB_SERVICES={"some-key": "some-val"}`))
		Expect(fakeExecutable.ExecuteCall.Receives.Execution.Env).To(ContainElement("CNB_STACK_ID=org.cloudfoundry.stacks.some-stack"))
	})

	when("the LOG_LEVEL environment variable is set", func() {
		logLevel := "debug"

		it.Before(func() {
			Expect(os.Setenv("LOG_LEVEL", logLevel)).To(Succeed())
		})

		it.After(func() {
			Expect(os.Unsetenv("LOG_LEVEL")).To(Succeed())
		})

		it("is passed to the lifecycle", func() {
			Expect(detector.Detect()).To(Succeed())

			Expect(installer.InstallCNBsCall.Receives.OrderFile).To(Equal(orderMetadata))
			Expect(installer.InstallCNBsCall.Receives.InstallDir).To(Equal(v3BuildpacksDir))

			Expect(installer.InstallLifecycleCall.Receives.Dst).To(Equal(v3LifecycleDir))

			Expect(fakeExecutable.ExecuteCall.Receives.Execution.Args).To(Equal([]string{
				"-app", v3AppDir,
				"-buildpacks", v3BuildpacksDir,
				"-order", orderMetadata,
				"-group", groupMetadata,
				"-plan", planMetadata,
				"-log-level", logLevel,
			}))
			Expect(fakeExecutable.ExecuteCall.Receives.Execution.Stderr).To(Equal(os.Stderr))
			Expect(fakeExecutable.ExecuteCall.Receives.Execution.Env).To(ContainElement(fmt.Sprintf(`LOG_LEVEL=%s`, logLevel)))
		})
	})

	when("v3-detector errors out", func() {
		it.Before(func() {
			fakeExecutable.ExecuteCall.Returns.Err = errors.New("failed to run v3 lifecycle detect")
		})

		it("returns error", func() {
			err := detector.Detect()
			Expect(err).To(MatchError("failed to run v3 lifecycle detect"))
		})
	})
}
