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

type StagingResultModifier interface {
	Modify(result builder.StagingResult) (builder.StagingResult, error)
}

type Uploader interface {
	Upload(path, url string) error
}

type Installer interface {
	Install() error
}

type Commander interface {
	Exec(cmd string, args ...string) (int, error)
}
