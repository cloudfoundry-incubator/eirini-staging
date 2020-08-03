package builder

import (
	"fmt"
)

const (
	Unknown        = "Unknown reason"
	DetectFailMsg  = "NoAppDetectedError"
	CompileFailMsg = "BuildpackCompileFailed"
	ReleaseFailMsg = "BuildpackReleaseFailed"

	FullDetectFailMsg      = "None of the buildpacks detected a compatible application"
	SupplyFailMsg          = "Failed to run all supply scripts"
	NoSupplyScriptFailMsg  = "Error: one of the buildpacks chosen to supply dependencies does not support multi-buildpack apps"
	MissingFinalizeWarnMsg = "Warning: the last buildpack is not compatible with multi-buildpack apps and cannot make use of any dependencies supplied by the buildpacks specified before it"
	FinalizeFailMsg        = "Failed to run finalize script"

	SystemFailCode   = 1
	DetectFailCode   = 222
	CompileFailCode  = 223
	ReleaseFailCode  = 224
	SupplyFailCode   = 225
	FinalizeFailCode = 227
)

type DescriptiveError struct {
	ExitCode   int
	InnerError error
	Message    string
}

func (e DescriptiveError) Error() string {
	if e.InnerError == nil {
		return fmt.Sprintf("%s: exit status %d", e.Message, e.ExitCode)
	}

	return fmt.Sprintf("%s: exit status %d - internal error: %s", e.Message, e.ExitCode, e.InnerError.Error())
}

func NewCompileFailError(err error) error {
	return DescriptiveError{Message: CompileFailMsg, ExitCode: CompileFailCode, InnerError: err}
}

func NewReleaseFailError(err error) error {
	return DescriptiveError{Message: ReleaseFailMsg, ExitCode: ReleaseFailCode, InnerError: err}
}

func NewSupplyFailError(err error) error {
	return DescriptiveError{Message: SupplyFailMsg, ExitCode: SupplyFailCode, InnerError: err}
}

func NewFinalizeFailError(err error) error {
	return DescriptiveError{Message: FinalizeFailMsg, ExitCode: FinalizeFailCode, InnerError: err}
}

func NewNoSupplyScriptFailError(err error) error {
	return DescriptiveError{Message: NoSupplyScriptFailMsg, ExitCode: SupplyFailCode, InnerError: err}
}
