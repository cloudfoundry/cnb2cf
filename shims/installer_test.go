package shims_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/cloudfoundry/cnb2cf/shims"
	"github.com/cloudfoundry/cnb2cf/shims/fakes"
	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/ansicleaner"
	"github.com/jarcoal/httpmock"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testInstaller(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect func(interface{}, ...interface{}) Assertion

		installer     *shims.CNBInstaller
		tmpDir        string
		buffer        *bytes.Buffer
		err           error
		fakeInstaller *fakes.DepInstaller
		manifest      *libbuildpack.Manifest
		logger        *libbuildpack.Logger
	)

	it.Before(func() {
		Expect = NewWithT(t).Expect

		Expect(os.Setenv("CF_STACK", "cflinuxfs3")).To(Succeed())
		httpmock.Reset()
		tmpDir, err = ioutil.TempDir("", "")
		Expect(err).ToNot(HaveOccurred())

		buffer = new(bytes.Buffer)

		logger = libbuildpack.NewLogger(ansicleaner.New(buffer))

		var err error
		manifest, err = libbuildpack.NewManifest(filepath.Join("testdata", "buildpack"), logger, time.Now())

		Expect(err).To(BeNil())
		fakeInstaller = &fakes.DepInstaller{}
		installer = shims.NewCNBInstaller(manifest, fakeInstaller)
	})

	it.After(func() {
		Expect(os.Unsetenv("CF_STACK")).To(Succeed())
		os.RemoveAll(tmpDir)
	})

	when("DownloadCNBs", func() {
		var buildpackTOML shims.BuildpackTOML

		it.Before(func() {
			buildpackTOML, err = shims.ParseBuildpackTOML(filepath.Join("testdata", "buildpack", "buildpack.toml"))
			Expect(err).NotTo(HaveOccurred())
		})

		when("installing the top level buildpacks", func() {
			var installedDeps []string
			it.Before(func() {
				installedDeps = []string{}
				fakeInstaller.InstallOnlyVersionCall.Stub = func(depName, path string) error {
					installedDeps = append(installedDeps, depName)
					return nil
				}
			})
			it("should install all the buildpacks", func() {
				paths, err := installer.DownloadCNBs(buildpackTOML, tmpDir)
				Expect(err).NotTo(HaveOccurred())

				// ordering can change due to map function
				Expect(installedDeps).To(ConsistOf("this.is.a.fake.bpA", "this.is.a.fake.bpB", "this.is.a.fake.bpC"))
				Expect(len(installedDeps)).To(Equal(3))

				correctNames := []string{"this.is.a.fake.bpA", "this.is.a.fake.bpB", "this.is.a.fake.bpC"}
				Expect(len(paths)).To(Equal(len(correctNames)))

				// Sort array that was once map keys, as it is unordered
				sort.Strings(paths)
				for idx, path := range paths {
					Expect(path).To(ContainSubstring(filepath.Join(tmpDir, correctNames[idx])))
				}
			})

			it("should not install already present buildpacks", func() {
				Expect(os.MkdirAll(filepath.Join(tmpDir, "this.is.a.fake.bpC", "1.0.2"), 0777)).To(Succeed())
				paths, err := installer.DownloadCNBs(buildpackTOML, tmpDir)
				Expect(err).NotTo(HaveOccurred())

				// ordering can change due to map function
				Expect(installedDeps).To(ConsistOf("this.is.a.fake.bpA", "this.is.a.fake.bpB"))
				Expect(len(installedDeps)).To(Equal(2))

				correctNames := []string{"this.is.a.fake.bpA", "this.is.a.fake.bpB"}
				Expect(len(paths)).To(Equal(len(correctNames)))

				// Sort array that was once map keys, as it is unordered
				sort.Strings(paths)
				for idx, path := range paths {
					Expect(path).To(ContainSubstring(filepath.Join(tmpDir, correctNames[idx])))
				}
			})
		})
	})

	when("InstallCNBs", func() {
		it.Before(func() {
			contents, err := ioutil.ReadFile(filepath.Join("testdata", "buildpack", "bpA.tgz"))
			Expect(err).ToNot(HaveOccurred())

			httpmock.RegisterResponder("GET", "https://a-fake-url.com/bpA.tgz", httpmock.NewStringResponder(200, string(contents)))

			contents, err = ioutil.ReadFile(filepath.Join("testdata", "buildpack", "bpB.tgz"))
			Expect(err).ToNot(HaveOccurred())

			httpmock.RegisterResponder("GET", "https://a-fake-url.com/bpB.tgz", httpmock.NewStringResponder(200, string(contents)))

			contents, err = ioutil.ReadFile(filepath.Join("testdata", "buildpack", "bp.tgz"))
			Expect(err).ToNot(HaveOccurred())

			httpmock.RegisterResponder("GET", "https://a-fake-url.com/bp.tgz", httpmock.NewStringResponder(200, string(contents)))

		})

		it.Before(func() {
			// reset tests to use an actuall installer :(
			realInstaller := libbuildpack.NewInstaller(manifest)
			Expect(err).To(BeNil())
			installer = shims.NewCNBInstaller(manifest, realInstaller)
		})

		it("installs the latest/unique buildpacks from an order.toml that are not already installed", func() {

			Expect(os.MkdirAll(filepath.Join(tmpDir, "this.is.a.fake.bpC", "1.0.2"), 0777)).To(Succeed())
			Expect(installer.InstallCNBs(filepath.Join("testdata", "buildpack", "buildpack.toml"), tmpDir)).To(Succeed())

			Expect(filepath.Join(tmpDir, "this.is.a.fake.bpA", "1.0.1", "a.txt")).To(BeAnExistingFile())
			Expect(filepath.Join(tmpDir, "this.is.a.fake.bpB", "1.0.2", "b.txt")).To(BeAnExistingFile())
			Expect(filepath.Join(tmpDir, "this.is.a.fake.bpA", "latest")).To(BeAnExistingFile())
			Expect(filepath.Join(tmpDir, "this.is.a.fake.bpB", "latest")).To(BeAnExistingFile())

			Expect(buffer.String()).ToNot(ContainSubstring("Installing this.is.a.fake.bpC"))
			Expect(filepath.Join(tmpDir, "this.is.a.fake.bpC")).To(BeADirectory())

		})

	})

	when("InstallLifecycle", func() {
		when("lifecyle version < 0.7.x", func() {
			it.Before(func() {
				// create a manifest, and a real installer

				realInstaller := libbuildpack.NewInstaller(manifest)
				Expect(err).To(BeNil())
				installer = shims.NewCNBInstaller(manifest, realInstaller)

				contents, err := ioutil.ReadFile(filepath.Join("testdata", "buildpack", "lifecycle-bundle.tgz"))
				Expect(err).ToNot(HaveOccurred())

				httpmock.RegisterResponder("GET", "https://a-fake-url.com/lifecycle-bundle.tgz",
					httpmock.NewStringResponder(200, string(contents)))
			})

			it("unpacks the lifecycle bundle and globs the contents of the subfolder and copies it somewhere", func() {
				Expect(installer.InstallLifecycle(tmpDir)).To(Succeed())
				keepBinaries := []string{"detector", "builder", "launcher"}

				for _, binary := range keepBinaries {
					Expect(filepath.Join(tmpDir, binary)).To(BeAnExistingFile())
				}
			})
		})

		when("lifecyle version >= 0.7.x", func() {
			// From lifecycle 0.7.x, they made a new executable named "lifecycle" and files
			// like detector, builder etc. are symlinks to this lifecycle executable
			it.Before(func() {
				// create a manifest, and a real installer
				manifest, err := libbuildpack.NewManifest(filepath.Join("testdata", "buildpack_lifecycle_v7"), logger, time.Now())
				Expect(err).To(BeNil())

				fakeInstaller = &fakes.DepInstaller{}
				installer = shims.NewCNBInstaller(manifest, fakeInstaller)

				realInstaller := libbuildpack.NewInstaller(manifest)
				Expect(err).To(BeNil())
				installer = shims.NewCNBInstaller(manifest, realInstaller)

				contents, err := ioutil.ReadFile(filepath.Join("testdata", "buildpack_lifecycle_v7", "lifecycle-bundle.tgz"))
				Expect(err).ToNot(HaveOccurred())

				httpmock.RegisterResponder("GET", "https://a-fake-url.com/lifecycle-bundle-v0.7.2.tgz",
					httpmock.NewStringResponder(200, string(contents)))
			})

			it("unpacks the lifecycle bundle and globs the contents of the subfolder and copies it somewhere", func() {
				Expect(installer.InstallLifecycle(tmpDir)).To(Succeed())
				keepBinaries := []string{"detector", "builder", "launcher", "lifecycle"}

				for _, binary := range keepBinaries {
					Expect(filepath.Join(tmpDir, binary)).To(BeAnExistingFile())
				}
			})
		})
	})
}
