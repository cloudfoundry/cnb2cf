package main

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/cloudfoundry/shim-generator/buildpack"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) != 4 {
		log.Fatal("Wrong number of args")
	}
	srcDir, destDir, _ := os.Args[1], os.Args[2], os.Args[3]

	err := os.Mkdir(destDir, 0777)
	if err != nil {
		log.Fatal("Failed to create directory")
	}

	if err := CopyDirectory(srcDir, destDir); err != nil {
		log.Fatal(err.Error())
	}

	buildpackTOML := buildpack.BuildpackTOML{}
	_, err = toml.DecodeFile(filepath.Join("cnbDir", "buildpack.toml"), &buildpackTOML)
	buildpackID := buildpackTOML.Buildpack.ID
	manifest := buildpack.Manifest{}
	manifestYAMLFile, err := os.OpenFile(filepath.Join(destDir, "manifest.yml"), os.O_RDWR, 0666)

	yaml.NewDecoder(manifestYAMLFile).Decode(&manifest)

	deps := manifest["dependencies"].([]interface{})
	firstDep := deps[0]
	firstDep.(map[interface{}]interface{})["name"] = buildpackID

}

func CopyDirectory(srcDir, destDir string) error {
	_, err := os.Stat(destDir)
	if os.IsNotExist(err) {
		return fmt.Errorf("destination dir must exist: %s", destDir)
	} else if err != nil {
		return err
	}
	_, err = os.Stat(srcDir)
	if os.IsNotExist(err) {
		return fmt.Errorf("source dir must exist: %s", srcDir)
	} else if err != nil {
		return err
	}

	files, err := ioutil.ReadDir(srcDir)
	if err != nil {
		return err
	}

	for _, f := range files {
		src := filepath.Join(srcDir, f.Name())
		dest := filepath.Join(destDir, f.Name())

		if m := f.Mode(); m&os.ModeSymlink != 0 {
			target, err := os.Readlink(src)
			if err != nil {
				return fmt.Errorf("Error while reading symlink '%s': %v", src, err)
			}
			if err := os.Symlink(target, dest); err != nil {
				return fmt.Errorf("Error while creating '%s' as symlink to '%s': %v", dest, target, err)
			}
		} else if f.IsDir() {
			err = os.MkdirAll(dest, f.Mode())
			if err != nil {
				return err
			}
			if err := CopyDirectory(src, dest); err != nil {
				return err
			}
		} else {
			rc, err := os.Open(src)
			if err != nil {
				return err
			}

			err = writeToFile(rc, dest, f.Mode())
			if err != nil {
				rc.Close()
				return err
			}
			rc.Close()
		}
	}

	return nil
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
