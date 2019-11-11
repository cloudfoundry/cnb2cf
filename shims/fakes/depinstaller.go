package fakes

import (
	"sync"

	"github.com/cloudfoundry/libbuildpack"
)

type DepInstaller struct {
	InstallDependencyCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			Dep       libbuildpack.Dependency
			OutputDir string
		}
		Returns struct {
			Error error
		}
		Stub func(libbuildpack.Dependency, string) error
	}
	InstallOnlyVersionCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			DepName    string
			InstallDir string
		}
		Returns struct {
			Error error
		}
		Stub func(string, string) error
	}
}

func (f *DepInstaller) InstallDependency(param1 libbuildpack.Dependency, param2 string) error {
	f.InstallDependencyCall.Lock()
	defer f.InstallDependencyCall.Unlock()
	f.InstallDependencyCall.CallCount++
	f.InstallDependencyCall.Receives.Dep = param1
	f.InstallDependencyCall.Receives.OutputDir = param2
	if f.InstallDependencyCall.Stub != nil {
		return f.InstallDependencyCall.Stub(param1, param2)
	}
	return f.InstallDependencyCall.Returns.Error
}
func (f *DepInstaller) InstallOnlyVersion(param1 string, param2 string) error {
	f.InstallOnlyVersionCall.Lock()
	defer f.InstallOnlyVersionCall.Unlock()
	f.InstallOnlyVersionCall.CallCount++
	f.InstallOnlyVersionCall.Receives.DepName = param1
	f.InstallOnlyVersionCall.Receives.InstallDir = param2
	if f.InstallOnlyVersionCall.Stub != nil {
		return f.InstallOnlyVersionCall.Stub(param1, param2)
	}
	return f.InstallOnlyVersionCall.Returns.Error
}
