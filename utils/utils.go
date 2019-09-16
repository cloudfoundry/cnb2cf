package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

func WriteToFile(source io.Reader, destFile string, mode os.FileMode) error {
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

func GetFilesFromZip(zipPath string) ([]string, error) {
	var result []string
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return []string{}, err
	}

	defer r.Close()

	for _, f := range r.File {
		result = append(result, f.Name)
	}
	return result, nil
}

func GetFileContentsFromZip(zipPath, innerFile string) ([]byte, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return []byte{}, err
	}

	defer r.Close()

	for _, f := range r.File {
		if f.Name == innerFile {
			fileReader, err := f.Open()
			if err != nil {
				return []byte{}, err
			}
			defer fileReader.Close()
			contents, err := ioutil.ReadAll(fileReader)
			if err != nil {
				return []byte{}, err
			}
			return contents, nil
		}
	}
	return []byte{}, fmt.Errorf("unable to find file %s", innerFile)
}
