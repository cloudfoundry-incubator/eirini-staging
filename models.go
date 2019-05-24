package eirinistaging

import bap "code.cloudfoundry.org/buildpackapplifecycle"

const (
	Unknown = "Unknown reason"
	// DetectFailMsg  = "Failed to detect buildpack"
	// CompileFailMsg = "Failed to compile droplet"
	// ReleaseFailMsg = "Failed to build droplet release"

	DetectFailMsg  = "NoAppDetectedError"
	CompileFailMsg = "BuildpackCompileFailed"
	ReleaseFailMsg = "BuildpackReleaseFailed"

	DETECT_FAIL_CODE  = 222
	COMPILE_FAIL_CODE = 223
	RELEASE_FAIL_CODE = 224
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
