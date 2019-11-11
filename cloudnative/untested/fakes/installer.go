package fakes

import "sync"

type Installer struct {
	DownloadCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			Uri         string
			Checksum    string
			Destination string
		}
		Returns struct {
			Error error
		}
		Stub func(string, string, string) error
	}
}

func (f *Installer) Download(param1 string, param2 string, param3 string) error {
	f.DownloadCall.Lock()
	defer f.DownloadCall.Unlock()
	f.DownloadCall.CallCount++
	f.DownloadCall.Receives.Uri = param1
	f.DownloadCall.Receives.Checksum = param2
	f.DownloadCall.Receives.Destination = param3
	if f.DownloadCall.Stub != nil {
		return f.DownloadCall.Stub(param1, param2, param3)
	}
	return f.DownloadCall.Returns.Error
}
