package shims_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/cnb2cf/shims"
	"github.com/cloudfoundry/cnb2cf/shims/fakes"
	"github.com/cloudfoundry/libbuildpack"

	"github.com/golang/mock/gomock"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=finalizer.go --destination=mocks_shims_test.go --package=shims_test
//go:generate mockgen -source=detector.go --destination=mocks_detector_shims_test.go --package=shims_test

func testFinalizer(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect func(interface{}, ...interface{}) Assertion

		finalizer       shims.Finalizer
		fakeExecutable  *fakes.Executable
		fakeEnvironment *fakes.Environment
		mockCtrl        *gomock.Controller
		mockDetector    *MockLifecycleDetectRunner
		tempDir,
		v2AppDir,
		v3AppDir,
		v2DepsDir,
		v2CacheDir,
		v3LayersDir,
		v3LauncherDir,
		v3BuildpacksDir,
		orderDir,
		orderMetadata,
		planMetadata,
		groupMetadata,
		profileDir,
		binDir,
		depsIndex string
		finalizeLogger *libbuildpack.Logger
	)

	it.Before(func() {
		Expect = NewWithT(t).Expect

		mockCtrl = gomock.NewController(t)
		mockDetector = NewMockLifecycleDetectRunner(mockCtrl)

		var err error
		tempDir, err = ioutil.TempDir("", "tmp")
		Expect(err).NotTo(HaveOccurred())

		v2AppDir = filepath.Join(tempDir, "v2_app")
		Expect(os.MkdirAll(v2AppDir, 0777)).To(Succeed())

		v3AppDir = filepath.Join(tempDir, "v3_app")
		Expect(os.MkdirAll(v3AppDir, 0777)).To(Succeed())

		v2DepsDir = filepath.Join(tempDir, "deps")

		depsIndex = "0"
		Expect(os.MkdirAll(filepath.Join(v2DepsDir, depsIndex), 0777)).To(Succeed())

		v2CacheDir = filepath.Join(tempDir, "cache")
		Expect(os.MkdirAll(tempDir, 0777)).To(Succeed())

		v3LayersDir = filepath.Join(tempDir, "layers")
		Expect(os.MkdirAll(v3LayersDir, 0777)).To(Succeed())

		v3LauncherDir = filepath.Join(tempDir, "launch")
		Expect(os.MkdirAll(v3LauncherDir, 0777)).To(Succeed())

		v3BuildpacksDir = filepath.Join(tempDir, "cnbs")
		Expect(os.MkdirAll(v3BuildpacksDir, 0777)).To(Succeed())

		orderDir = filepath.Join(tempDir, "order")
		Expect(os.MkdirAll(orderDir, 0777)).To(Succeed())

		orderMetadata = filepath.Join(tempDir, "order.toml")
		planMetadata = filepath.Join(tempDir, "plan.toml")
		groupMetadata = filepath.Join(tempDir, "group.toml")

		profileDir = filepath.Join(tempDir, "profile")
		Expect(os.MkdirAll(profileDir, 0777)).To(Succeed())

		binDir = filepath.Join(tempDir, "bin")
		Expect(os.MkdirAll(binDir, 0777)).To(Succeed())

		Expect(os.Setenv("CF_STACK", "some-stack")).To(Succeed())

		finalizeLogger = &libbuildpack.Logger{}

		fakeExecutable = &fakes.Executable{}
		fakeEnvironment = &fakes.Environment{}

		finalizer = shims.Finalizer{
			V2AppDir:        v2AppDir,
			V3AppDir:        v3AppDir,
			V2DepsDir:       v2DepsDir,
			V2CacheDir:      v2CacheDir,
			V3LayersDir:     v3LayersDir,
			V3LauncherDir:   v3LauncherDir,
			V3BuildpacksDir: v3BuildpacksDir,
			DepsIndex:       depsIndex,
			OrderDir:        orderDir,
			OrderMetadata:   orderMetadata,
			PlanMetadata:    planMetadata,
			GroupMetadata:   groupMetadata,
			ProfileDir:      profileDir,
			V3LifecycleDir:  binDir,
			Detector:        mockDetector,
			Logger:          finalizeLogger,
			Executable:      fakeExecutable,
			Environment:     fakeEnvironment,
		}
	})

	it.After(func() {
		mockCtrl.Finish()
		Expect(os.Unsetenv("CF_STACK")).To(Succeed())
		Expect(os.RemoveAll(tempDir)).To(Succeed())
	})

	when("RunLifecycleBuild", func() {
		it.Before(func() {
			fakeEnvironment.StackCall.Returns.String = "some-stack"
			fakeEnvironment.ServicesCall.Returns.String = `{"some-key": "some-val"}`
		})

		it("when executing lifecycle binary", func() {
			Expect(finalizer.RunLifecycleBuild()).To(Succeed())

			Expect(fakeExecutable.ExecuteCall.Receives.Args).To(Equal([]string{
				"-app", v3AppDir,
				"-buildpacks", v3BuildpacksDir,
				"-group", groupMetadata,
				"-layers", v3LayersDir,
				"-plan", planMetadata,
			}))

			Expect(fakeExecutable.ExecuteCall.Receives.Options.Stdout).To(Equal(os.Stdout))
			Expect(fakeExecutable.ExecuteCall.Receives.Options.Stderr).To(Equal(os.Stderr))

			env := fakeExecutable.ExecuteCall.Receives.Options.Env
			Expect(env).To(ContainElement(`CNB_SERVICES={"some-key": "some-val"}`))
			Expect(env).To(ContainElement("CNB_STACK_ID=org.cloudfoundry.stacks.some-stack"))
		})

		when("the lifecycle build binary fails", func() {
			it.Before(func() {
				fakeExecutable.ExecuteCall.Returns.Err = errors.New("lifecycle build phase failed")
			})

			it("returns an error", func() {
				err := finalizer.RunLifecycleBuild()
				Expect(err).To(MatchError("lifecycle build phase failed"))
			})
		})
	})

	when("GenerateOrderTOML", func() {
		it("should write a order.toml file with metabuildpack id's and versions", func() {
			orderFileA := filepath.Join(orderDir, "orderA.toml")
			orderFileB := filepath.Join(orderDir, "orderB.toml")

			Expect(ioutil.WriteFile(orderFileA, []byte(`
api = "0.2"
[buildpack]
id = "org.some-org.first-buildpack.shimmed"
name = "First Buildpack"
version = "1.2.3"

[[order]]
[[order.group]]
id = "org.some-org.first-buildpack"
version = "1.2.3"
`), os.ModePerm)).To(Succeed())
			Expect(ioutil.WriteFile(orderFileB, []byte(`
api = "0.2"
[buildpack]
id = "org.some-org.second-buildpack.shimmed"
name = "Second Buildpack"
version = "4.5.6"

[[order]]
[[order.group]]
id = "org.some-org.second-buildpack"
version = "4.5.6"
`), os.ModePerm)).To(Succeed())

			// Get all of the buildpack.toml files from OrderDir

			// Parse each one into a "buildpack"
			// Put buildpack id + version into a new order.toml
			// write the order.toml file
			Expect(finalizer.GenerateOrderTOML()).To(Succeed())

			Expect(finalizer.OrderMetadata).To(BeAnExistingFile())
			buildpack, err := shims.ParseBuildpackTOML(finalizer.OrderMetadata)
			Expect(err).ToNot(HaveOccurred())

			Expect(buildpack.Order).To(HaveLen(1))
			Expect(buildpack.Order[0].Groups).To(HaveLen(2))
			firstBP := buildpack.Order[0].Groups[0]
			secondBP := buildpack.Order[0].Groups[1]

			Expect(firstBP.ID).To(Equal("org.some-org.first-buildpack.shimmed"))
			Expect(firstBP.Version).To(Equal("1.2.3"))
			Expect(secondBP.ID).To(Equal("org.some-org.second-buildpack.shimmed"))
			Expect(secondBP.Version).To(Equal("4.5.6"))
		})
	})

	when("RunV3Detect", func() {
		it("runs detection when group or plan metadata does not exist", func() {
			mockDetector.
				EXPECT().
				RunLifecycleDetect()
			Expect(finalizer.RunV3Detect()).To(Succeed())
		})

		it("does NOT run detection when group and plan metadata exists", func() {
			Expect(ioutil.WriteFile(groupMetadata, []byte(""), 0666)).To(Succeed())
			Expect(ioutil.WriteFile(planMetadata, []byte(""), 0666)).To(Succeed())

			mockDetector.
				EXPECT().
				RunLifecycleDetect().
				Times(0)
			Expect(finalizer.RunV3Detect()).To(Succeed())
		})
	})

	when("IncludePreviousV2Buildpacks", func() {
		var createDirs, createFiles []string

		it.Before(func() {
			Expect(os.RemoveAll(filepath.Join(v2DepsDir, depsIndex))).To(Succeed())

			depsIndex = "2"
			finalizer.DepsIndex = depsIndex
			Expect(os.MkdirAll(filepath.Join(v2DepsDir, depsIndex), 0777)).To(Succeed())

			createDirs = []string{"bin", "lib"}
			createFiles = []string{"config.yml"}
			for _, dir := range createDirs {
				Expect(os.MkdirAll(filepath.Join(v2DepsDir, "0", dir), 0777)).To(Succeed())
			}

			for _, file := range createFiles {
				Expect(ioutil.WriteFile(filepath.Join(v2DepsDir, "0", file), []byte(file), 0666)).To(Succeed())
			}

			Expect(ioutil.WriteFile(groupMetadata, []byte(`[[group]]
  id = "buildpack.1"
  version = "0.0.1"
[[group]]
  id = "buildpack.2"
  version = "0.0.1"`), 0666)).To(Succeed())
			Expect(ioutil.WriteFile(planMetadata, []byte(""), 0666)).To(Succeed())
		})

		it("copies v2 layers and metadata where v3 lifecycle expects them for build and launch", func() {
			// not failing if a layer has already been moved
			Expect(finalizer.IncludePreviousV2Buildpacks()).To(Succeed())

			// putting the v2 layers in the corrent directory structure
			for _, dir := range createDirs {
				Expect(filepath.Join(v3LayersDir, "buildpack.0", "layer", dir)).To(BeADirectory())
			}

			for _, file := range createFiles {
				Expect(filepath.Join(v3LayersDir, "buildpack.0", "layer", file)).To(BeAnExistingFile())
			}

			// writing the group metadata in the order the buildpacks should be sourced
			groupMetadataContents, err := ioutil.ReadFile(groupMetadata)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(groupMetadataContents)).To(Equal(`[[group]]
  id = "buildpack.0"
  version = "0.0.1"

[[group]]
  id = "buildpack.1"
  version = "0.0.1"

[[group]]
  id = "buildpack.2"
  version = "0.0.1"
`))
		})
	})

	when("RestoreV3Cache", func() {
		it.Before(func() {
			cloudfoundryV3Cache := filepath.Join(v2CacheDir, "cnb")
			testLayers := filepath.Join(cloudfoundryV3Cache, "org.cloudfoundry.generic.buildpack")
			Expect(os.MkdirAll(cloudfoundryV3Cache, 0777)).To(Succeed())
			Expect(os.MkdirAll(filepath.Join(testLayers, "layer"), 0777)).To(Succeed())
			Expect(os.MkdirAll(filepath.Join(testLayers, "anotherLayer"), 0777)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(testLayers, "anotherLayer", "cachedContents"), []byte("cached contents"), 0666)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(testLayers, "anotherLayer", "anotherLayer.toml"), []byte("cache=true"), 0666)).To(Succeed())
		})

		it("should restore cache before building", func() {
			restoredLayers := filepath.Join(finalizer.V3LayersDir, "org.cloudfoundry.generic.buildpack")
			Expect(finalizer.RestoreV3Cache()).ToNot(HaveOccurred())
			Expect(filepath.Join(restoredLayers, "layer")).To(BeADirectory())
			Expect(filepath.Join(restoredLayers, "anotherLayer")).To(BeADirectory())
			Expect(filepath.Join(restoredLayers, "anotherLayer", "cachedContents")).To(BeAnExistingFile())
			contents, err := ioutil.ReadFile(filepath.Join(restoredLayers, "anotherLayer", "cachedContents"))
			Expect(err).ToNot(HaveOccurred())
			Expect(contents).To(ContainSubstring("cached contents"))
		})
	})

	when("MoveV3Layers", func() {
		it.Before(func() {
			Expect(os.MkdirAll(filepath.Join(v3LayersDir, "config"), 0777)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(v3LayersDir, "config", "metadata.toml"), []byte(""), 0666)).To(Succeed())

			Expect(os.MkdirAll(filepath.Join(v3LayersDir, "layers"), 0777)).To(Succeed())
			Expect(os.MkdirAll(filepath.Join(v3LayersDir, "anotherLayers", "innerLayer"), 0777)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(v3LayersDir, "anotherLayers", "innerLayer.toml"), []byte("cache=true"), 0666)).To(Succeed())
		})

		it("moves the layers to deps dir and metadata to build dir", func() {
			Expect(finalizer.MoveV3Layers()).To(Succeed())
			Expect(filepath.Join(v2AppDir, ".cloudfoundry", "metadata.toml")).To(BeAnExistingFile())
			Expect(filepath.Join(v2DepsDir, "layers")).To(BeAnExistingFile())
			Expect(filepath.Join(v2DepsDir, "anotherLayers")).To(BeAnExistingFile())
		})

		it("copies cacheable layers to the cache/cnb directory", func() {
			Expect(filepath.Join(v2CacheDir, "cnb")).ToNot(BeADirectory())
			Expect(finalizer.MoveV3Layers()).To(Succeed())
			Expect(filepath.Join(v2CacheDir, "cnb", "anotherLayers", "innerLayer")).To(BeADirectory())
		})
	})

	when("MoveV2Layers", func() {
		it("moves directories and creates the dst dir if it doesn't exist", func() {
			Expect(finalizer.MoveV2Layers(filepath.Join(v2DepsDir, depsIndex), filepath.Join(v3LayersDir, "buildpack.0", "layers.0"))).To(Succeed())
			Expect(filepath.Join(v3LayersDir, "buildpack.0", "layers.0")).To(BeADirectory())
		})
	})

	when("RenameEnvDir", func() {
		it("renames the env dir to env.build", func() {
			Expect(os.Mkdir(filepath.Join(v3LayersDir, "env"), 0777)).To(Succeed())
			Expect(finalizer.RenameEnvDir(v3LayersDir)).To(Succeed())
			Expect(filepath.Join(v3LayersDir, "env.build")).To(BeADirectory())
		})

		it("does nothing when the env dir does NOT exist", func() {
			Expect(finalizer.RenameEnvDir(v3LayersDir)).To(Succeed())
			Expect(filepath.Join(v3LayersDir, "env.build")).NotTo(BeADirectory())
		})
	})

	when("UpdateGroupTOML", func() {
		it.Before(func() {
			Expect(os.RemoveAll(filepath.Join(v2DepsDir, depsIndex))).To(Succeed())

			depsIndex = "1"
			finalizer.DepsIndex = depsIndex
			Expect(os.MkdirAll(filepath.Join(v2DepsDir, depsIndex), 0777)).To(Succeed())

			Expect(ioutil.WriteFile(groupMetadata, []byte(`[[group]]
	 id = "org.cloudfoundry.buildpacks.nodejs"
	 version = "0.0.2"
	[[group]]
	 id = "org.cloudfoundry.buildpacks.npm"
	 version = "0.0.3"`), 0777)).To(Succeed())
		})

		it("adds v2 buildpacks to the group.toml", func() {
			Expect(finalizer.UpdateGroupTOML("buildpack.0")).To(Succeed())
			groupMetadataContents, err := ioutil.ReadFile(groupMetadata)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(groupMetadataContents)).To(Equal(`[[group]]
  id = "buildpack.0"
  version = "0.0.1"

[[group]]
  id = "org.cloudfoundry.buildpacks.nodejs"
  version = "0.0.2"

[[group]]
  id = "org.cloudfoundry.buildpacks.npm"
  version = "0.0.3"
`))
		})
	})

	when("In V3 Layers Dir", func() {
		var (
			testLayers            string
			Dep1LayerMetadataPath string
			Dep2LayerMetadataPath string
		)

		it.Before(func() {
			testLayers = filepath.Join(finalizer.V3LayersDir, "org.cloudfoundry.generic.buildpack")
			Expect(os.MkdirAll(testLayers, os.ModePerm)).To(Succeed())

			Dep1LayerMetadataPath = filepath.Join(testLayers, "dep1.toml")
			Dep2LayerMetadataPath = filepath.Join(testLayers, "dep2.toml")
			Dep1LayerPath := filepath.Join(testLayers, "dep1")
			Dep2LayerPath := filepath.Join(testLayers, "dep2")

			Expect(os.MkdirAll(Dep2LayerPath, os.ModePerm)).To(Succeed())
			Expect(os.MkdirAll(Dep1LayerPath, os.ModePerm)).To(Succeed())

			Expect(ioutil.WriteFile(Dep1LayerMetadataPath, []byte(`launch = true
			build = false
			cache = true

			[metadata]
			extradata = "shamoo"`), 0777)).To(Succeed())

			Expect(ioutil.WriteFile(Dep2LayerMetadataPath, []byte(`launch = true
			build = true
			cache = true
			[metadata]
			extradata = "shamwow"`), 0777)).To(Succeed())
		})

		it("can read layer.toml", func() {
			dep1Metadata, err := finalizer.ReadLayerMetadata(Dep1LayerMetadataPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(dep1Metadata.Launch).To(Equal(true))
			Expect(dep1Metadata.Build).To(Equal(false))
			Expect(dep1Metadata.Cache).To(Equal(true))

			dep2Metadata, err := finalizer.ReadLayerMetadata(Dep2LayerMetadataPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(dep2Metadata.Launch).To(Equal(true))
			Expect(dep2Metadata.Build).To(Equal(true))
			Expect(dep2Metadata.Cache).To(Equal(true))
		})

		it("can move layer to cache if needed", func() {
			Expect(finalizer.MoveV3Layers()).To(Succeed())

			layersCacheDir := filepath.Join(v2CacheDir, "cnb", "org.cloudfoundry.generic.buildpack")
			Expect(filepath.Join(layersCacheDir, "dep1")).To(BeADirectory())
			Expect(filepath.Join(layersCacheDir, "dep2")).To(BeADirectory())

		})
	})

	when("AddFakeCNBBuildpack", func() {
		it("adds the v2 buildpack as a no-op cnb buildpack", func() {
			Expect(os.Setenv("CF_STACK", "cflinuxfs3")).To(Succeed())
			Expect(finalizer.AddFakeCNBBuildpack("buildpack.0")).To(Succeed())

			buildpackTOMLPath := filepath.Join(v3BuildpacksDir, "buildpack.0", "0.0.1", "buildpack.toml")
			buildpackTOML, err := ioutil.ReadFile(buildpackTOMLPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(buildpackTOML)).To(Equal(`[buildpack]
  id = "buildpack.0"
  name = "buildpack.0"
  version = "0.0.1"

[[stacks]]
  id = "org.cloudfoundry.stacks.cflinuxfs3"
`))

			Expect(filepath.Join(v3BuildpacksDir, "buildpack.0", "0.0.1", "bin", "build")).To(BeAnExistingFile())
		})
	})

	when("WriteProfileLaunch", func() {
		it("writes a profile script that execs the v3 launcher", func() {
			Expect(finalizer.WriteProfileLaunch()).To(Succeed())
			contents, err := ioutil.ReadFile(filepath.Join(profileDir, shims.V3LaunchScript))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(Equal(fmt.Sprintf(`export CNB_STACK_ID="org.cloudfoundry.stacks.%s"
export CNB_LAYERS_DIR="$DEPS_DIR"
export CNB_APP_DIR="$HOME"
exec $HOME/.cloudfoundry/%s "$2"
`, os.Getenv("CF_STACK"), shims.V3Launcher)))
		})
	})
}

type bp struct {
	id       string
	optional bool
}
