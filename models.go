package eirinistaging

import (
	"code.cloudfoundry.org/eirini-staging/builder"
)

type NotZipFileError struct {
	err error
}

func (z NotZipFileError) Error() string {
	return z.err.Error()
}

type Executor interface {
	ExecuteRecipe() error
}

//go:generate counterfeiter . StagingResultModifier
type StagingResultModifier interface {
	Modify(result builder.StagingResult) (builder.StagingResult, error)
}

//go:generate counterfeiter . Uploader
type Uploader interface {
	Upload(path, url string) error
}

//go:generate counterfeiter . Installer
type Installer interface {
	Install() error
}

//go:generate counterfeiter . Commander
type Commander interface {
	Exec(cmd string, args ...string) (int, error)
}
