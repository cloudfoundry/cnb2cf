/*
 * Copyright 2019-2020 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
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
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	templ "text/template"

	"github.com/Masterminds/semver"
	"github.com/fatih/color"

	buildpackBp "github.com/buildpack/libbuildpack/buildpack"
	layersBp "github.com/buildpack/libbuildpack/layers"
	loggerBp "github.com/buildpack/libbuildpack/logger"
	"github.com/cloudfoundry/libcfbuildpack/buildpack"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"github.com/cloudfoundry/libcfbuildpack/logger"
)

const (
	DefaultDstDir    = "packaged-cnb"
	DefaultCacheBase = ".cnb-packager-cache"
)

var identityColor = color.New(color.FgBlue)

type Packager struct {
	buildpack       buildpack.Buildpack
	layers          layers.Layers
	logger          logger.Logger
	outputDirectory string
}

func New(bpDir, outputDir, version, cacheDir string) (Packager, error) {
	l, err := loggerBp.DefaultLogger("")
	if err != nil {
		return Packager{}, err
	}

	if err := insertTemplateVersion(bpDir, version); err != nil {
		return Packager{}, err
	}

	specBP, err := buildpackBp.New(bpDir, l)
	if err != nil {
		return Packager{}, err
	}

	log := logger.Logger{Logger: l}
	b := buildpack.NewBuildpack(specBP, log)

	depCache, err := filepath.Abs(filepath.Join(cacheDir, buildpack.CacheRoot))
	if err != nil {
		return Packager{}, err
	}

	return Packager{
		b,
		layers.NewLayers(layersBp.NewLayers(depCache, l), layersBp.NewLayers(depCache, l), b, log),
		log,
		outputDir,
	}, nil
}

type pkgFile struct {
	path        string
	packagePath string
}

func (p Packager) Create(cache bool) error {
	p.logger.Title(p.buildpack)

	if err := p.prePackage(); err != nil {
		return err
	}

	includedFiles, err := p.buildpack.IncludeFiles()
	if err != nil {
		return err
	}

	var allFiles []pkgFile
	for _, i := range includedFiles {
		path, err := filepath.Abs(filepath.Join(p.buildpack.Root, i))
		if err != nil {
			return err
		}
		f := pkgFile{
			path:        path,
			packagePath: i,
		}
		allFiles = append(allFiles, f)
	}

	if cache {
		dependencyFiles, err := p.cacheDependencies()
		if err != nil {
			return err
		}
		allFiles = append(allFiles, dependencyFiles...)
	}

	return p.createPackage(allFiles)
}

func insertTemplateVersion(bpDir, version string) error {
	bpTomlPath := filepath.Join(bpDir, "buildpack.toml")
	v := struct {
		Version string
	}{
		Version: version,
	}

	template, err := templ.ParseFiles(bpTomlPath)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(bpTomlPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to open buildpack.toml : %s", err))
	}

	return template.Execute(file, v)
}

func (p Packager) cacheDependencies() ([]pkgFile, error) {
	var files []pkgFile

	deps, err := p.buildpack.Dependencies()
	if err != nil {
		return nil, err
	}

	for _, dep := range deps {
		p.logger.Header("Caching %s", p.prettyIdentity(dep))

		layer := p.layers.DownloadLayer(dep)

		a, err := layer.Artifact()
		if err != nil {
			return nil, err
		}

		f := pkgFile{
			path:        a,
			packagePath: filepath.Join(buildpack.CacheRoot, dep.SHA256, filepath.Base(a)),
		}

		metaF := pkgFile{
			path:        layer.Metadata,
			packagePath: filepath.Join(buildpack.CacheRoot, dep.SHA256+".toml"),
		}

		files = append(files, f, metaF)
	}

	return files, nil
}

func (Packager) prettyIdentity(v logger.Identifiable) string {
	if v == nil {
		return ""
	}

	name, description := v.Identity()

	if description == "" {
		return identityColor.Sprint(name)
	}

	return identityColor.Sprintf("%s %s", name, description)
}

func (p Packager) Archive() error {
	defer os.RemoveAll(p.outputDirectory)
	fileName := filepath.Base(p.outputDirectory)
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
	if !info.Mode().IsRegular() && !info.Mode().IsDir() {
		return nil
	}

	if header, err := tar.FileInfoHeader(info, path); err == nil {
		header.Name = stripBaseDirectory(p.outputDirectory, path)

		if header.Name == "" {
			return nil
		}

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if info.Mode().IsRegular() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			if _, err := io.Copy(tw, file); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p Packager) createPackage(files []pkgFile) error {
	if len(files) == 0 {
		return errors.New("no files included")
	}

	p.logger.Header("Creating package in %s", p.outputDirectory)

	for _, file := range files {
		p.logger.Body("Adding %s", file.packagePath)
		outputDir := filepath.Dir(filepath.Join(p.outputDirectory, file.packagePath))
		if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
			return err
		}
		if err := helper.CopyFile(file.path, filepath.Join(p.outputDirectory, file.packagePath)); err != nil {
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

	p.logger.Header("Pre-Package with %s", strings.Join(cmd.Args, " "))

	return cmd.Run()
}

func stripBaseDirectory(base, path string) string {
	return strings.TrimPrefix(strings.Replace(path, base, "", -1), string(filepath.Separator))
}

func (p Packager) Summary() (string, error) {
	var out string
	if err := p.depsSummary(&out); err != nil {
		return "", err
	}

	p.defaultsSummary(&out)
	p.stacksSummary(&out)

	return out, nil
}

func (p Packager) depsSummary(out *string) error {

	type depKey struct {
		Idx     int
		ID      string
		Version string
	}

	bpMetadata := p.buildpack.Metadata
	deps, ok := bpMetadata["dependencies"].([]map[string]interface{})
	if !ok || len(deps) == 0 {
		return nil
	}

	*out = "\nPackaged binaries:\n\n"
	*out += "| name | version | stacks |\n|-|-|-|\n"

	depMap := map[depKey]buildpack.Stacks{}
	for _, d := range deps {
		dep, err := buildpack.NewDependency(d)
		if err != nil {
			return err
		}
		depKey := depKey{
			ID:      dep.ID,
			Version: dep.Version.Version.String(),
		}
		if _, ok := depMap[depKey]; !ok {
			depMap[depKey] = dep.Stacks
		} else {
			depMap[depKey] = append(depMap[depKey], dep.Stacks...)
		}
	}
	depKeyArray := make([]depKey, 0)
	for key, _ := range depMap {
		depKeyArray = append(depKeyArray, key)
	}

	sort.SliceStable(depKeyArray, func(i, j int) bool {
		alph := strings.Compare(depKeyArray[i].ID, depKeyArray[j].ID)
		if alph < 0 {
			return true
		} else if alph == 0 {
			versionI, err := semver.NewVersion(depKeyArray[i].Version)
			if err != nil {
				return false
			}
			versionJ, err := semver.NewVersion(depKeyArray[j].Version)
			if err != nil {
				return false
			}
			return versionI.GreaterThan(versionJ)
		}
		return false
	})

	for _, dKey := range depKeyArray {
		stacks := depMap[dKey]
		stackStringArray := []string{}
		for _, stack := range stacks {
			stackStringArray = append(stackStringArray, string(stack))
		}
		*out += fmt.Sprintf("| %s | %s | %s |\n", dKey.ID, dKey.Version, strings.Join(stackStringArray, ", "))
	}

	return nil
}

func (p Packager) defaultsSummary(out *string) {
	bpMetadata := p.buildpack.Metadata
	defaults, ok := bpMetadata[buildpack.DefaultVersions].(map[string]interface{})
	if !ok {
		return
	}

	if len(defaults) > 0 {
		*out += "\nDefault binary versions:\n\n"
		*out += "| name | version |\n|-|-|\n"
		for name, version := range defaults {
			*out += fmt.Sprintf("| %s | %s |\n", name, version)
		}
	}
}

func (p Packager) stacksSummary(out *string) {
	if len(p.buildpack.Stacks) < 1 {
		return
	}

	*out += `
Supported stacks:

| name |
|-|
`
	for _, stack := range p.buildpack.Stacks {
		*out += fmt.Sprintf("| %s |\n", stack.ID)
	}
}
