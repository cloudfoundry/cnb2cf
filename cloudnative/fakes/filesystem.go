package fakes

import "sync"

type Filesystem struct {
	ReadFileCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			Name string
		}
		Returns struct {
			ByteSlice []byte
			Error     error
		}
		Stub func(string) ([]byte, error)
	}
}

func (f *Filesystem) ReadFile(param1 string) ([]byte, error) {
	f.ReadFileCall.Lock()
	defer f.ReadFileCall.Unlock()
	f.ReadFileCall.CallCount++
	f.ReadFileCall.Receives.Name = param1
	if f.ReadFileCall.Stub != nil {
		return f.ReadFileCall.Stub(param1)
	}
	return f.ReadFileCall.Returns.ByteSlice, f.ReadFileCall.Returns.Error
}
