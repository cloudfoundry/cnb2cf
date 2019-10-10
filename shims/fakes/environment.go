package fakes

import "sync"

type Environment struct {
	ServicesCall struct {
		sync.Mutex
		CallCount int
		Returns   struct {
			String string
		}
		Stub func() string
	}
	StackCall struct {
		sync.Mutex
		CallCount int
		Returns   struct {
			String string
		}
		Stub func() string
	}
}

func (f *Environment) Services() string {
	f.ServicesCall.Lock()
	defer f.ServicesCall.Unlock()
	f.ServicesCall.CallCount++
	if f.ServicesCall.Stub != nil {
		return f.ServicesCall.Stub()
	}
	return f.ServicesCall.Returns.String
}
func (f *Environment) Stack() string {
	f.StackCall.Lock()
	defer f.StackCall.Unlock()
	f.StackCall.CallCount++
	if f.StackCall.Stub != nil {
		return f.StackCall.Stub()
	}
	return f.StackCall.Returns.String
}
