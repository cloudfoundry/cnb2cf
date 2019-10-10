package fakes

import "sync"

type Installer struct {
	InstallCNBsCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			OrderFile  string
			InstallDir string
		}
		Returns struct {
			Error error
		}
		Stub func(string, string) error
	}
	InstallLifecycleCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			Dst string
		}
		Returns struct {
			Error error
		}
		Stub func(string) error
	}
}

func (f *Installer) InstallCNBs(param1 string, param2 string) error {
	f.InstallCNBsCall.Lock()
	defer f.InstallCNBsCall.Unlock()
	f.InstallCNBsCall.CallCount++
	f.InstallCNBsCall.Receives.OrderFile = param1
	f.InstallCNBsCall.Receives.InstallDir = param2
	if f.InstallCNBsCall.Stub != nil {
		return f.InstallCNBsCall.Stub(param1, param2)
	}
	return f.InstallCNBsCall.Returns.Error
}
func (f *Installer) InstallLifecycle(param1 string) error {
	f.InstallLifecycleCall.Lock()
	defer f.InstallLifecycleCall.Unlock()
	f.InstallLifecycleCall.CallCount++
	f.InstallLifecycleCall.Receives.Dst = param1
	if f.InstallLifecycleCall.Stub != nil {
		return f.InstallLifecycleCall.Stub(param1)
	}
	return f.InstallLifecycleCall.Returns.Error
}
