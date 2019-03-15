package shimmer

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/cloudfoundry/cnb2cf/buildpackdata"
	_ "github.com/cloudfoundry/cnb2cf/statik"
	"github.com/rakyll/statik/fs"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

type Shimmer struct {
	SrcDir  string
	DestDir string
	Lang    string
}

func (s *Shimmer) Shim() error {
	if err := s.copyTemplate(); err != nil {
		log.Fatal("Failed to copy template: ", err.Error())
	}

	if err := os.Mkdir(s.DestDir, 0777); err != nil {
		log.Fatal("Failed to create directory: ", err.Error())
	}

	//if err := copyDirectory(s.TemplatePath, s.DestDir); err != nil {
	//	log.Fatal("Failed to copy template: ", err.Error())
	//}

	if err := s.updateManifestYAML(); err != nil {
		log.Fatal("Failed to generate manifest.yml: ", err.Error())
	}

	if err := s.generateOrderTOML(); err != nil {
		log.Fatal("Failed to generate order.toml: ", err.Error())
	}
	return nil
}

func (s *Shimmer) copyTemplate() error {
	statikFS, err := fs.New()
	if err != nil {
		return err
	}

	if err := fs.Walk(statikFS, "/", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		file, err := statikFS.Open(path)
		if os.IsNotExist(err) {
			return nil
		}else if err != nil {
			return err
		}
		return s.copy(path, file, s.DestDir)
	}); err != nil {
		return err
	}

	return nil
}

func (s *Shimmer) copy(srcPath string, file http.File, destPath string) error {
	//if info.IsDir() {
	//	if err := os.MkdirAll(filepath.Join(destPath, srcPath), 0666); err != nil {
	//		return err
	//	}
	//} else {
	//	//if err := writeToFile(srcPath,)
	//}
	return nil
}

func copyDirectory(srcDir, destDir string) error {
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
			if err := copyDirectory(src, dest); err != nil {
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

func (s *Shimmer) updateManifestYAML() error {
	bpTOML := buildpackdata.BuildpackTOML{}
	if _, err := toml.DecodeFile(filepath.Join(s.SrcDir, "buildpack.toml"), &bpTOML); err != nil {
		return err
	}

	bpManifestPath := filepath.Join(s.DestDir, "manifest.yml.template")
	manifestYAMLFile, err := os.OpenFile(bpManifestPath, os.O_RDWR, 0666)
	if err != nil {
		return err
	}

	defer manifestYAMLFile.Close()

	bpManifest := buildpackdata.Metadata{}
	if err := yaml.NewDecoder(manifestYAMLFile).Decode(&bpManifest); err != nil {
		return err
	}

	bpManifest.Language = s.Lang
	bpManifest.Dependencies = append([]buildpackdata.Dependency{{
		Name:    bpTOML.Buildpack.ID,
		Version: bpTOML.Buildpack.Version,
		SHA256:  "< FILL THIS OUT >",
		URI:     "< FILL THIS OUT >",
		Stacks:  []string{"< FILL THIS OUT >"},
	}}, bpManifest.Dependencies...)

	if err := manifestYAMLFile.Truncate(0); err != nil {
		return err
	}

	if _, err := manifestYAMLFile.Seek(0, 0); err != nil {
		return err
	}

	return yaml.NewEncoder(manifestYAMLFile).Encode(bpManifest)
}

func (s *Shimmer) generateOrderTOML() error {
	bpTOML := buildpackdata.BuildpackTOML{}
	if _, err := toml.DecodeFile(filepath.Join(s.SrcDir, "buildpack.toml"), &bpTOML); err != nil {
		return err
	}

	orderTOML := buildpackdata.OrderTOML{
		Groups: []buildpackdata.OrderGroup{
			{
				Labels: []string{s.Lang},
				Buildpacks: []buildpackdata.OrderBuildpack{
					{
						ID:      bpTOML.Buildpack.ID,
						Version: "latest",
					},
				},
			},
		},
	}

	orderTOMLFile, err := os.Create(filepath.Join(s.DestDir, "order.toml"))
	if err != nil {
		return err
	}
	defer orderTOMLFile.Close()

	return toml.NewEncoder(orderTOMLFile).Encode(orderTOML)
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
