/*
 * Copyright 2018-2019 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cnbpackager

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	buildpackBp "github.com/buildpack/libbuildpack/buildpack"
	layersBp "github.com/buildpack/libbuildpack/layers"
	loggerBp "github.com/buildpack/libbuildpack/logger"
	"github.com/cloudfoundry/libcfbuildpack/buildpack"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"github.com/cloudfoundry/libcfbuildpack/logger"
)

type Packager struct {
	buildpack       buildpack.Buildpack
	layers          layers.Layers
	logger          logger.Logger
	outputDirectory string
}


func DefaultPackager(outputDirectory string) (Packager, error) {
	l, err := loggerBp.DefaultLogger("")
	if err != nil {
		return Packager{}, err
	}

	logger := logger.Logger{Logger: l}

	b, err := buildpackBp.DefaultBuildpack(l)
	if err != nil {
		return Packager{}, err
	}
	buildpack := buildpack.NewBuildpack(b, logger)
	layers := layers.NewLayers(layersBp.NewLayers(buildpack.CacheRoot, l), layersBp.NewLayers(buildpack.CacheRoot, l), buildpack, logger)

	return Packager{
		buildpack,
		layers,
		logger,
		outputDirectory,
	}, nil
}

func New(bpDir, outputDir string) (Packager, error) {
	l, err := loggerBp.DefaultLogger("")
	if err != nil {
		return Packager{}, err
	}
	specBP, err := buildpackBp.New(bpDir, l)
	if err != nil {
		return Packager{}, err
	}

	log := logger.Logger{Logger: l}
	b := buildpack.NewBuildpack(specBP, log)

	return Packager{
		b,
		layers.NewLayers(layersBp.NewLayers(b.CacheRoot, l), layersBp.NewLayers(b.CacheRoot, l), b, log),
		log,
		outputDir,
	}, nil
}

func (p Packager) Create(cache bool) error {
	p.logger.FirstLine("Packaging %s", p.logger.PrettyIdentity(p.buildpack))

	if err := p.prePackage(); err != nil {
		return err
	}

	includedFiles, err := p.buildpack.IncludeFiles()
	if err != nil {
		return err
	}

	var dependencyFiles []string
	if cache {
		dependencyFiles, err = p.cacheDependencies()
		if err != nil {
			return err
		}
		includedFiles = append(includedFiles, dependencyFiles...)
	}

	return p.createPackage(includedFiles)
}

func (p Packager) cacheDependencies() ([]string, error) {
	var files []string

	deps, err := p.buildpack.Dependencies()
	if err != nil {
		return nil, err
	}

	for _, dep := range deps {
		p.logger.FirstLine("Caching %s", p.logger.PrettyIdentity(dep))

		layer := p.layers.DownloadLayer(dep)

		a, err := layer.Artifact()
		if err != nil {
			return nil, err
		}

		artifact, err := filepath.Rel(p.buildpack.Root, a)
		if err != nil {
			return nil, err
		}

		metadata, err := filepath.Rel(p.buildpack.Root, layer.Metadata)
		if err != nil {
			return nil, err
		}

		files = append(files, artifact, metadata)
	}

	return files, nil
}

func (p Packager) Archive(cached bool) error {
	defer os.RemoveAll(p.outputDirectory)
	fileName := filepath.Base(p.outputDirectory)
	if cached {
		fileName = fileName + "-cached"
	}
	tarFile := filepath.Join(filepath.Dir(p.outputDirectory), fileName+".tgz")

	file, err := os.Create(tarFile)
	if err != nil {
		return err
	}
	defer file.Close()

	gw := gzip.NewWriter(file)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	filepath.Walk(p.outputDirectory, func(path string, info os.FileInfo, err error) error {
		return p.addTarFile(tw, info, path)
	})

	return nil
}

func (p Packager) addTarFile(tw *tar.Writer, info os.FileInfo, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	if header, err := tar.FileInfoHeader(info, path); err == nil {
		header.Name = stripBaseDirectory(p.outputDirectory, path)

		if !info.Mode().IsRegular() {
			return nil
		}

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if _, err := io.Copy(tw, file); err != nil {
			return err
		}

	}
	return nil
}

func (p Packager) createPackage(files []string) error {
	p.logger.FirstLine("Creating package in %s", p.outputDirectory)

	for _, file := range files {
		p.logger.SubsequentLine("Adding %s", file)
		if err := helper.CopyFile(filepath.Join(p.buildpack.Root, file), filepath.Join(p.outputDirectory, file)); err != nil {
			return err
		}
	}
	return nil
}

func (p Packager) prePackage() error {
	pp, ok := p.buildpack.PrePackage()
	if !ok {
		return nil
	}

	cmd := exec.Command(pp)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = p.buildpack.Root

	p.logger.FirstLine("Pre-Package with %s", strings.Join(cmd.Args, " "))

	return cmd.Run()
}

func stripBaseDirectory(base, path string) string {
	return strings.TrimPrefix(strings.Replace(path, base, "", -1), string(filepath.Separator))
}
