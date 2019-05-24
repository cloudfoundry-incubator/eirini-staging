package eirinistaging

import bap "code.cloudfoundry.org/buildpackapplifecycle"

const (
	Unknown        = "Unknown reason"
	DetectFailMsg  = "NoAppDetectedError"
	CompileFailMsg = "BuildpackCompileFailed"
	ReleaseFailMsg = "BuildpackReleaseFailed"

	DetectFailCode  = 222
	CompileFailCode = 223
	ReleaseFailCode = 224
)

type ErrorWithExitCode struct {
	ExitCode   int
	InnerError error
}

func (e ErrorWithExitCode) Error() string {
	return e.InnerError.Error()
}

type Executor interface {
	ExecuteRecipe() error
}

//go:generate counterfeiter . StagingResultModifier
type StagingResultModifier interface {
	Modify(result bap.StagingResult) (bap.StagingResult, error)
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
