package cloudnative

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
)

type DependencyInstaller struct{}

func NewDependencyInstaller() DependencyInstaller {
	return DependencyInstaller{}
}

func (di DependencyInstaller) Download(uri, checksum, destination string) error {
	err := os.MkdirAll(filepath.Dir(destination), 0755)
	if err != nil {
		return err
	}

	output, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer output.Close()

	u, err := url.Parse(uri)
	if err != nil {
		return err
	}

	var source io.ReadCloser

	if u.Scheme == "file" {
		source, err = os.Open(u.Path)
		if err != nil {
			return err
		}
		defer source.Close()
	} else {
		client := http.Client{}
		req, err := http.NewRequest("GET", uri, nil)
		if err != nil {
			return err
		}
		gitToken := os.Getenv("GITHUB_TOKEN")
		if gitToken != "" {
			req.Header["Authorization"] = []string{"token " + gitToken}
		}
		response, err := client.Do(req)
		if err != nil {
			return err
		}
		defer response.Body.Close()
		source = response.Body

		if response.StatusCode < 200 || response.StatusCode > 299 {
			return fmt.Errorf("could not download: %d", response.StatusCode)
		}
	}

	_, err = io.Copy(output, source)

	if err != nil {
		return err
	}

	if err := libbuildpack.CheckSha256(destination, checksum); err != nil {
		return err
	}

	return nil
}

func (di DependencyInstaller) Copy(source, destination string) error {
	info, err := os.Stat(source)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("source %s is not a directory", source)
	}

	if err := os.MkdirAll(destination, 0755); err != nil {
		return err
	}

	if err := libbuildpack.CopyDirectory(source, destination); err != nil {
		return err
	}

	return nil
}
