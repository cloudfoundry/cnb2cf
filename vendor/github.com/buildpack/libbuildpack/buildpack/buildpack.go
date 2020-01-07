/*
 * Copyright 2018-2020 the original author or authors.
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

package buildpack

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/buildpack/libbuildpack/internal"
	"github.com/buildpack/libbuildpack/logger"
)

// Buildpack represents the metadata associated with a buildpack.
type Buildpack struct {
	// Info is information about the buildpack.
	Info Info `toml:"buildpack"`

	// Metadata is the additional metadata included in the buildpack.
	Metadata Metadata `toml:"metadata"`

	// Root is the path to the root directory for the buildpack.
	Root string

	// Stacks is the collection of stacks that the buildpack supports.
	Stacks []Stack `toml:"stacks"`

	logger logger.Logger
}

// New creates an instance of Buildpack given a root dir and a logger extracting the contents of the buildpack.toml file in the root
// of the buildpack.
func New(rootDir string, logger logger.Logger) (Buildpack, error) {
	f, err := ioutil.ReadFile(filepath.Join(rootDir, "buildpack.toml"))
	if err != nil {
		return Buildpack{}, fmt.Errorf("could not find buildpack.toml in the directory %s", rootDir)
	}

	b := Buildpack{Root: rootDir, logger: logger}
	if err := toml.Unmarshal(f, &b); err != nil {
		return Buildpack{}, err
	}

	logger.Debug("Buildpack: %#v", b)
	return b, nil
}

// DefaultBuildpack creates a new instance of Buildpack extracting the contents of the buildpack.toml file in the root
// of the buildpack.
func DefaultBuildpack(logger logger.Logger) (Buildpack, error) {
	f, err := findBuildpackTOML()
	if err != nil {
		return Buildpack{}, err
	}

	in, err := os.Open(f)
	if err != nil {
		return Buildpack{}, err
	}
	defer in.Close()

	b := Buildpack{Root: filepath.Dir(f), logger: logger}

	if _, err := toml.DecodeReader(in, &b); err != nil {
		return Buildpack{}, err
	}

	logger.Debug("Buildpack: %#v", b)
	return b, nil
}

func findBuildpackTOML() (string, error) {
	path, err := internal.Argument(0)
	if err != nil {
		return "", err
	}

	dir, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return "", err
	}

	for {
		if dir == "/" {
			return "", fmt.Errorf("could not find buildpack.toml in the directory hierarchy")
		}

		f := filepath.Join(dir, "buildpack.toml")
		if exist, err := internal.FileExists(f); err != nil {
			return "", err
		} else if exist {
			return f, nil
		}

		dir, err = filepath.Abs(filepath.Join(dir, ".."))
		if err != nil {
			return "", err
		}
	}
}
