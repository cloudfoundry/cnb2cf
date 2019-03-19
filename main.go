package main

//go:generate statik -src=./template

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/cloudfoundry/cnb2cf/shimmer"
	"github.com/pkg/errors"
)

const expectedArgs = 1

func main() {
	actualArgs := len(os.Args) - 1
	if actualArgs != expectedArgs {
		log.Fatalf("Wrong number of arguments, expected %d got %d", expectedArgs, actualArgs)
	}

	configPath := os.Args[1]

	if err := run(configPath); err != nil {
		log.Fatal(err)
	}
}

func run(configPath string) error {
	config, err := shimmer.LoadConfig(configPath)
	if err != nil {
		return errors.Wrap(err, "Failed to load config")
	}

	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		return errors.Wrap(err, "Failed to create tempdir")
	}

	defer os.RemoveAll(tempDir)

	outputDir := "."

	if err := shimmer.CreateBuildpack(config, tempDir); err != nil {
		return errors.Wrap(err, "Failed to convert buildpack to shim")
	}

	return shimmer.CreateZip(config, tempDir, outputDir)
}
