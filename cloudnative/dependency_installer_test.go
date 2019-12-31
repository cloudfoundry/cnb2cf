package cloudnative_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/cloudfoundry/cnb2cf/cloudnative"
	"github.com/jarcoal/httpmock"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testDependencyInstaller(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect func(interface{}, ...interface{}) Assertion

		tmpDir      string
		destination string
		installer   cloudnative.DependencyInstaller
	)

	it.Before(func() {
		Expect = NewWithT(t).Expect

		var err error
		tmpDir, err = ioutil.TempDir("", "dependency-installer")
		Expect(err).NotTo(HaveOccurred())

		installer = cloudnative.NewDependencyInstaller()
	})

	it.After(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	when("Download", func() {
		var uri, checksum string

		it.Before(func() {
			httpmock.Activate()
			httpmock.RegisterResponder("GET", "https://example.com/uri-dependency.tgz", httpmock.NewStringResponder(200, "dependency-contents"))
			httpmock.RegisterResponder("GET", "https://example.com/garbage-uri.tgz", httpmock.NewStringResponder(500, ""))

			uri = "https://example.com/uri-dependency.tgz"
			checksum = "f058c8bf6b65b829e200ef5c2d22fde0ee65b96c1fbd1b88869be133aafab64a"
			destination = filepath.Join(tmpDir, "destination", "archive.tgz")
		})

		it.After(func() {
			httpmock.DeactivateAndReset()
		})

		it("installs the remote dependency source", func() {
			err := installer.Download(uri, checksum, destination)
			Expect(err).NotTo(HaveOccurred())

			contents, err := ioutil.ReadFile(destination)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(Equal("dependency-contents"))
		})

		when("GIT_TOKEN env var is set", func() {
			var prevAuthToken string
			var tokenVal string
			it.Before(func() {
				// what should this actually be???

				tokenVal = "some-auth-token"
				prevAuthToken = os.Getenv("GIT_TOKEN")
				Expect(os.Setenv("GIT_TOKEN", tokenVal)).To(Succeed())

				httpmock.RegisterResponder("GET", "https://example.com/uri-dependency.tgz", func(request *http.Request) (response *http.Response, e error) {
					contents := request.Header["Authorization"]
					res := new(http.Response)
					expectedToken := []string{"token " +tokenVal}
					if !reflect.DeepEqual(expectedToken, contents) {
						return res, fmt.Errorf("unexpected header value '%s' vs '%s'", contents, expectedToken)
					}
					res.StatusCode = 200
					res.Body = ioutil.NopCloser(strings.NewReader("dependency-contents"))
					res.Request = request
					return res, nil
				})
			})

			it.After(func() {
				Expect(os.Setenv("GIT_TOKEN", prevAuthToken)).To(Succeed())
			})
			it("uses correct auth to grab dependency", func() {
				err := installer.Download(uri, checksum, destination)
				Expect(err).NotTo(HaveOccurred())

				contents, err := ioutil.ReadFile(destination)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(contents)).To(Equal("dependency-contents"))
			})
		})

		when("the dependency cannot be downloaded", func() {
			it("returns an error", func() {
				err := installer.Download("https://example.com/garbage-uri.tgz", checksum, destination)
				Expect(err).To(MatchError(ContainSubstring("could not download")))
			})
		})

		when("the checksum does not match", func() {
			it("returns an error", func() {
				err := installer.Download(uri, "garbage-checksum", destination)
				Expect(err).To(MatchError(ContainSubstring("dependency sha256 mismatch")))
			})
		})
	})

	when("Copy", func() {
		var source, tmpFile string

		it.Before(func() {
			source = filepath.Join(tmpDir, "source", "dependency")
			destination = filepath.Join(tmpDir, "destination", "dependency")

			err := os.MkdirAll(source, 0755)
			Expect(err).NotTo(HaveOccurred())

			file, err := ioutil.TempFile(source, "copy")
			Expect(err).NotTo(HaveOccurred())
			defer file.Close()

			tmpFile = file.Name()
		})

		it("copies the source directory to the destination", func() {
			err := installer.Copy(source, destination)
			Expect(err).NotTo(HaveOccurred())
			Expect(filepath.Join(destination, filepath.Base(tmpFile))).To(BeAnExistingFile())
		})

		when("the destination cannot be created", func() {
			it.Before(func() {
				Expect(os.Chmod(tmpDir, 0000)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(tmpDir, 0755)).To(Succeed())
			})

			it("returns an error", func() {
				err := installer.Copy(source, destination)
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		when("the source files cannot be copied", func() {
			it.Before(func() {
				Expect(os.Chmod(tmpFile, 0000)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(tmpFile, 0644)).To(Succeed())
			})

			it("returns an error", func() {
				err := installer.Copy(source, destination)
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		when("the source does not exist", func() {
			it("returns an error", func() {
				err := installer.Copy("/no/such/path", destination)
				Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
			})
		})

		when("the source is not a directory", func() {
			it("returns an error", func() {
				err := installer.Copy(tmpFile, destination)
				Expect(err).To(MatchError(fmt.Sprintf("source %s is not a directory", tmpFile)))
			})
		})
	})
}
