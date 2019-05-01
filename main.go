package main

//go:generate statik -src=./template

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/cnb2cf/shimmer"
	"github.com/pkg/errors"
)

var expectedArgsMin = 1
var expectedArgsMax = 2

func main() {
	actualArgs := len(os.Args) - 1
	if actualArgs > expectedArgsMax || actualArgs < expectedArgsMin {
		log.Fatalf("wrong number of arguments, expected %d - %d got %d", expectedArgsMin, expectedArgsMax, actualArgs)
	}

	command := os.Args[1]

	if command == "build" {
		if err := build(); err != nil {
			log.Fatal(err)
		}
	} else {
		if err := run(command); err != nil {
			log.Fatal(err)
		}

	}

}

func downloadFile(url, destFile string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("could not download: %d", resp.StatusCode)
	}

	return writeToFile(resp.Body, destFile, 0666)
}

func writeToFile(source io.Reader, destFile string, mode os.FileMode) error {
	err := os.MkdirAll(filepath.Dir(destFile), 0755)
	if err != nil {
		return err
	}

	fh, err := os.OpenFile(destFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer fh.Close()

	_, err = io.Copy(fh, source)
	if err != nil {
		return err
	}

	return nil
}


func build() error {
	manifest, err := shimmer.LoadManifest()
	if err != nil {
		return errors.Wrap(err, "failed to load manifest")
	}

	tempDir, err := ioutil.TempDir("", "")
	if err != nil{
		return errors.Wrap(err, "failed to create tempDir")
	}

	for _,d := range manifest.Dependencies{
		sourceUri := d.SourceURI
		if sourceUri == ""{
			continue
		}else{
			if err := downloadFile(sourceUri,filepath.Join(tempDir, filepath.Base(sourceUri))); err != nil{
				return errors.Wrap(err, "failed to download required dependency")
			}
		}
	}

	return nil
}

func run(configPath string) error {
	config, err := shimmer.LoadConfig(configPath)
	if err != nil {
		return errors.Wrap(err, "failed to load config")
	}

	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		return errors.Wrap(err, "failed to create tempdir")
	}

	defer os.RemoveAll(tempDir)

	outputDir := "."

	if err := shimmer.CreateBuildpack(config, tempDir); err != nil {
		return errors.Wrap(err, "failed to convert buildpack to shim")
	}

	return shimmer.CreateZip(config, tempDir, outputDir)
}
