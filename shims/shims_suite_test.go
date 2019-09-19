package shims_test

import (
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitShims(t *testing.T) {
	suite := spec.New("shims", spec.Report(report.Terminal{}))

	suite("Detector", testDetector)
	suite("Finalizer", testFinalizer)
	suite("Installer", testInstaller)
	suite("Releaser", testReleaser)
	suite("Supplier", testSupplier)

	suite.Before(func(t *testing.T) {
		httpmock.Activate()
	})

	suite.After(func(t *testing.T) {
		httpmock.DeactivateAndReset()
	})

	suite.Run(t)
}
