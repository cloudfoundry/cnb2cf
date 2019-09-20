package shims

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
)

type Installer interface {
	InstallCNBs(orderFile string, installDir string) error
	InstallLifecycle(dst string) error
}

type Detector struct {
	V3LifecycleDir string

	AppDir string

	V3BuildpacksDir string

	OrderMetadata string
	GroupMetadata string
	PlanMetadata  string

	Installer Installer
}

func (d Detector) Detect() error {
	if err := d.Installer.InstallCNBs(d.OrderMetadata, d.V3BuildpacksDir); err != nil {
		return errors.Wrap(err, "failed to install buildpacks for detection")
	}

	return d.RunLifecycleDetect()
}

func (d Detector) RunLifecycleDetect() error {
	if err := d.Installer.InstallLifecycle(d.V3LifecycleDir); err != nil {
		return errors.Wrap(err, "failed to install v3 lifecycle binaries")
	}

	cmd := exec.Command(
		filepath.Join(d.V3LifecycleDir, V3Detector),
		"-app", d.AppDir,
		"-buildpacks", d.V3BuildpacksDir,
		"-order", d.OrderMetadata,
		"-group", d.GroupMetadata,
		"-plan", d.PlanMetadata,
	)

	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "CNB_STACK_ID=org.cloudfoundry.stacks."+os.Getenv("CF_STACK"))
	return cmd.Run()
}
