package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/cnb2cf/shims"
	"github.com/cloudfoundry/libbuildpack"
)

func main() {
	var logger = libbuildpack.NewLogger(os.Stdout)
	if len(os.Args) != 2 {
		logger.Error("incorrect number of arguments")
		os.Exit(1)
	}

	releaser := shims.Releaser{
		MetadataPath: filepath.Join(os.Args[1], ".cloudfoundry", "metadata.toml"),
		Writer:       os.Stdout,
	}

	if err := releaser.Release(); err != nil {
		log.Printf("Failed release step: %s\n", err)
		os.Exit(1)
	}
}
