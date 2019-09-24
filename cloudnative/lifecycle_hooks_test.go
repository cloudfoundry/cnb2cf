package cloudnative_test

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/cnb2cf/cloudnative"
	"github.com/cloudfoundry/cnb2cf/cloudnative/fakes"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testLifecycleHooks(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect func(interface{}, ...interface{}) Assertion

		tmpDir         string
		filesystem     *fakes.Filesystem
		lifecycleHooks cloudnative.LifecycleHooks
	)

	it.Before(func() {
		Expect = NewWithT(t).Expect

		var err error
		tmpDir, err = ioutil.TempDir("", "dependency-installer")
		Expect(err).NotTo(HaveOccurred())

		filesystem = &fakes.Filesystem{}
		filesystem.ReadFileCall.Returns.ByteSlice = []byte("contents")

		lifecycleHooks = cloudnative.NewLifecycleHooks(filesystem)
	})

	it.After(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	when("Install", func() {
		it("installs the given v2 lifecycle hook into the given directory", func() {
			for _, file := range []string{"compile", "detect", "finalize", "release", "supply"} {
				err := lifecycleHooks.Install(file, tmpDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(filepath.Join(tmpDir, "bin", file)).To(BeAnExistingFile())

				Expect(filesystem.ReadFileCall.Receives.Name).To(Equal(filepath.Join("/bin", file)))
			}
		})

		when("the filesystem cannot find a file with that name", func() {
			it.Before(func() {
				filesystem.ReadFileCall.Returns.Error = errors.New("failed to read file")
			})

			it("returns an error", func() {
				err := lifecycleHooks.Install("unknown", tmpDir)
				Expect(err).To(MatchError("could not install hook \"unknown\": failed to read file"))
			})
		})

		when("the bin directory cannot be created", func() {
			it.Before(func() {
				Expect(os.Chmod(tmpDir, 0000)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(tmpDir, 0755)).To(Succeed())
			})

			it("returns an error", func() {
				err := lifecycleHooks.Install("detect", tmpDir)
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		when("the hook already exists and is not writable", func() {
			it.Before(func() {
				err := os.MkdirAll(filepath.Join(tmpDir, "bin"), os.ModePerm)
				Expect(err).NotTo(HaveOccurred())

				_, err = os.OpenFile(filepath.Join(tmpDir, "bin", "detect"), os.O_CREATE, 0000)
				Expect(err).NotTo(HaveOccurred())
			})

			it.After(func() {
				Expect(os.Chmod(filepath.Join(tmpDir, "bin", "detect"), 0644)).To(Succeed())
			})

			it("returns an error", func() {
				err := lifecycleHooks.Install("detect", tmpDir)
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})
	})
}
