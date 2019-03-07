package buildpack

type BuildpackTOML struct {
	Buildpack struct{
		ID string `toml:"id"`
	}
}
