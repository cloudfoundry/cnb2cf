package packager

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/cnb2cf/cloudnative"
	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libcfbuildpack/packager/cnbpackager"
	"github.com/cloudfoundry/packit/cargo"
	"github.com/cloudfoundry/packit/cargo/jam/commands"
	"github.com/cloudfoundry/packit/pexec"
	"github.com/cloudfoundry/packit/scribe"
	"github.com/pkg/errors"

	_ "github.com/cloudfoundry/cnb2cf/statik"
)

var DefaultCacheDir = filepath.Join(os.Getenv("HOME"), ".cnb2cf", "cache")

func ExtractCNBSource(dep cloudnative.BuildpackMetadataDependency, src, dstDir string) error {
	if strings.HasSuffix(dep.Source, "/") {
		return libbuildpack.CopyDirectory(src, dstDir)
	}

	if strings.HasSuffix(dep.Source, ".zip") {
		return libbuildpack.ExtractZip(src, dstDir)
	}

	if strings.HasSuffix(dep.Source, ".tar.xz") {
		return libbuildpack.ExtractTarXz(src, dstDir)
	}

	return libbuildpack.ExtractTarGz(src, dstDir)
}

func BuildCNB(extractDir, outputDir string, cached bool, version string) (string, string, error) {
	foundSrc, err := FindCNB(extractDir)
	if err != nil {
		return "", "", err
	}

	path := fmt.Sprintf("%s.tgz", outputDir)

	_, err = os.Stat(filepath.Join(foundSrc, ".packit"))
	if err != nil {
		// RUN packager
		usr, err := user.Current()
		if err != nil {
			log.Fatal(err)
		}

		globalCacheDir := filepath.Join(usr.HomeDir, cnbpackager.DefaultCacheBase)

		packager, err := cnbpackager.New(foundSrc, outputDir, version, globalCacheDir)
		if err != nil {
			return "", "", err
		}

		if err := packager.Create(cached); err != nil {
			return "", "", err
		}

		if err := packager.Archive(); err != nil {
			return "", "", err
		}
	} else {
		// RUN jam pack
		logger := scribe.NewLogger(os.Stdout)
		bash := pexec.NewExecutable("bash")

		transport := cargo.NewTransport()
		directoryDuplicator := cargo.NewDirectoryDuplicator()
		buildpackParser := cargo.NewBuildpackParser()
		fileBundler := cargo.NewFileBundler()
		tarBuilder := cargo.NewTarBuilder(logger)
		prePackager := cargo.NewPrePackager(bash, logger, scribe.NewWriter(os.Stdout, scribe.WithIndent(2)))
		dependencyCacher := cargo.NewDependencyCacher(transport, logger)
		command := commands.NewPack(directoryDuplicator, buildpackParser, prePackager, dependencyCacher, fileBundler, tarBuilder, os.Stdout)

		args := []string{
			"--buildpack", filepath.Join(foundSrc, "buildpack.toml"),
			"--output", path,
			"--version", version,
		}

		if cached {
			args = append(args, "--offline")
		}

		err = command.Execute(args)
		if err != nil {
			return "", "", err
		}
	}

	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	hash := sha256.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		panic(err)
	}

	return path, hex.EncodeToString(hash.Sum(nil)), nil
}

// FindCNB returns the path to the cnb source if it can find a single buildpack.toml
// in the top level dir or within one directory
// This is to support source tar files with a root directory (github release structure)
func FindCNB(extractDir string) (string, error) {
	buildpackTOML := filepath.Join(extractDir, "buildpack.toml")
	if _, err := os.Stat(buildpackTOML); err == nil {
		return filepath.Dir(buildpackTOML), nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	paths, err := filepath.Glob(filepath.Join(extractDir, "*", "buildpack.toml"))
	if err != nil {
		return "", err
	}

	if len(paths) < 1 {
		return "", errors.New("failed to find find cnb source: no buildpack.toml")
	}

	if len(paths) > 1 {
		return "", errors.New("failed to find find cnb source: found multiple buildpack.toml files")
	}

	return filepath.Dir(paths[0]), nil
}
