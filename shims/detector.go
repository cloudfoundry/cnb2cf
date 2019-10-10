package shims

import (
	"fmt"
	"os"
	"sort"

	"github.com/cloudfoundry/libbuildpack/cutlass/execution"
	"github.com/cloudfoundry/libbuildpack/cutlass/glow"

	"github.com/pkg/errors"
)

//go:generate faux --interface Environment --output fakes/environment.go
type Environment interface {
	Services() string
	Stack() string
}

//go:generate faux --interface Installer --output fakes/installer.go
type Installer interface {
	InstallCNBs(orderFile, installDir string) error
	InstallLifecycle(dst string) error
}

type Detector struct {
	V3LifecycleDir string

	AppDir string

	V3BuildpacksDir string

	OrderMetadata string
	GroupMetadata string
	PlanMetadata  string

	Installer   Installer
	Environment Environment
	Executor    glow.Executable
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

	env := os.Environ()

	vcapServices := d.Environment.Services()
	env = append(env, fmt.Sprintf("CNB_SERVICES=%s", vcapServices))

	stack := d.Environment.Stack()
	env = append(env, fmt.Sprintf("CNB_STACK_ID=org.cloudfoundry.stacks.%s", stack))

	sort.Strings(env)

	args := []string{
		"-app", d.AppDir,
		"-buildpacks", d.V3BuildpacksDir,
		"-order", d.OrderMetadata,
		"-group", d.GroupMetadata,
		"-plan", d.PlanMetadata,
	}
	_, _, err := d.Executor.Execute(execution.Options{
		Stderr: os.Stderr,
		Env:    env,
	}, args...)
	if err != nil {
		return err
	}

	return nil
}
