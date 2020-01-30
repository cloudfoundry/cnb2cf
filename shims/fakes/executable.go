package fakes

import (
	"sync"

	"github.com/cloudfoundry/packit/pexec"
)

type Executable struct {
	ExecuteCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			Execution pexec.Execution
		}
		Returns struct {
			Stdout string
			Stderr string
			Err    error
		}
		Stub func(pexec.Execution) (string, string, error)
	}
}

func (f *Executable) Execute(param1 pexec.Execution) (string, string, error) {
	f.ExecuteCall.Lock()
	defer f.ExecuteCall.Unlock()
	f.ExecuteCall.CallCount++
	f.ExecuteCall.Receives.Execution = param1
	if f.ExecuteCall.Stub != nil {
		return f.ExecuteCall.Stub(param1)
	}
	return f.ExecuteCall.Returns.Stdout, f.ExecuteCall.Returns.Stderr, f.ExecuteCall.Returns.Err
}
