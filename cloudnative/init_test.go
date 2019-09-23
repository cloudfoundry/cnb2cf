package cloudnative_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitCloudNative(t *testing.T) {
	suite := spec.New("cloudnative", spec.Report(report.Terminal{}))

	suite("Buildpack", testBuildpack)

	suite.Run(t)
}
