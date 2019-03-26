package shimmer

type BuildpackTOML struct {
	Buildpack CNBBuildpack           `toml:"buildpack"`
	Metadata  map[string]interface{} `toml:"metadata"`
	Stacks    []struct {
		ID          string   `toml:"id"`
		Mixins      []string `toml:"mixins"`
		BuildImages []string `toml:"build-images"`
		RunImages   []string `toml:"run-images"`
	} `toml:"stacks"`
}

type OrderTOML struct {
	Groups []CNBGroup `toml:"groups" yaml:"groups"`
}

type CNBGroup struct {
	Buildpacks []CNBBuildpack `toml:"buildpacks" yaml:"buildpacks"`
}

type CNBBuildpack struct {
	ID       string `toml:"id" yaml:"id"`
	Name     string `toml:"name,omitempty"`
	Version  string `toml:"version" yaml:"version"`
	Optional bool   `toml:"optional,omitempty" yaml:"optional,omitempty"`
}

type ManifestYAML struct {
	Language     string         `yaml:"language"`
	PrePackage   string         `yaml:"pre_package"`
	IncludeFiles []string       `yaml:"include_files"`
	Dependencies []V2Dependency `yaml:"dependencies"`
	Stack        string         `yaml:"stack"`
}

type V2Dependency struct {
	Name     string   `yaml:"name"`
	Version  string   `yaml:"version"`
	URI      string   `yaml:"uri"`
	SHA256   string   `yaml:"sha256"`
	CFStacks []string `yaml:"cf_stacks"`
}
