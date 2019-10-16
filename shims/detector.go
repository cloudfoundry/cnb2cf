package shims

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/packit"
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

//go:generate faux --interface Executable --output fakes/executable.go
type Executable interface {
	Execute(packit.Execution) (stdout, stderr string, err error)
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
	Executor    Executable
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
	var logger = libbuildpack.NewLogger(os.Stderr)
	logger.Info("Environ from detect in cnb2cf")
	env := os.Environ()

	sort.Strings(env)

	logger.Info(strings.Join(env, "\n"))

	vcapServices := d.Environment.Services()
	env = append(env, fmt.Sprintf("CNB_SERVICES=%s", vcapServices))

	stack := d.Environment.Stack()
	env = append(env, fmt.Sprintf("CNB_STACK_ID=org.cloudfoundry.stacks.%s", stack))

	_, _, err := d.Executor.Execute(packit.Execution{
		Args: []string{
			"-app", d.AppDir,
			"-buildpacks", d.V3BuildpacksDir,
			"-order", d.OrderMetadata,
			"-group", d.GroupMetadata,
			"-plan", d.PlanMetadata,
		},
		Stderr: os.Stderr,
		Env:    env,
	})
	if err != nil {
		return err
	}

	return nil
}
