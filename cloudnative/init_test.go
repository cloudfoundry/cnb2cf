package cloudnative_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitCloudNative(t *testing.T) {
	suite := spec.New("cloudnative", spec.Report(report.Terminal{}))

	suite("Buildpack", testBuildpack)
	suite("Manifest", testManifest)
	suite("DependencyInstaller", testDependencyInstaller)
	suite("LifecycleHooks", testLifecycleHooks)
	suite("Filesystem", testFilesystem)
	suite("Environment", testEnvironment)

	suite.Run(t)
}
