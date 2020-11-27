package types

type Ctx struct {
	TargetName string
	SrcDir     string
	BuildDir   string
}

type Project struct {
	Name    string
	Version string
}
