package builder

import (
	"fmt"
)

const (
	Unknown        = "Unknown reason"
	DetectFailMsg  = "NoAppDetectedError"
	CompileFailMsg = "BuildpackCompileFailed"
	ReleaseFailMsg = "BuildpackReleaseFailed"

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

type ProcessTypes map[string]string

type Buildpack struct {
	Name       string `json:"name"`
	Key        string `json:"key"`
	URL        string `json:"url"`
	SkipDetect bool   `json:"skip_detect,omit_empty"`
}

type BuildpackMetadata struct {
	Key     string `json:"key"`
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type LifecycleMetadata struct {
	BuildpackKey      string              `json:"buildpack_key,omitempty"`
	DetectedBuildpack string              `json:"detected_buildpack"`
	Buildpacks        []BuildpackMetadata `json:"buildpacks"`
}

type StagingResult struct {
	LifecycleMetadata `json:"lifecycle_metadata"`
	ProcessTypes      `json:"process_types"`
	ExecutionMetadata string `json:"execution_metadata"`
	LifecycleType     string `json:"lifecycle_type"`
}

func NewStagingResult(procTypes ProcessTypes, lifeMeta LifecycleMetadata) StagingResult {
	return StagingResult{
		LifecycleType:     "buildpack",
		LifecycleMetadata: lifeMeta,
		ProcessTypes:      procTypes,
		ExecutionMetadata: "",
	}
}

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

func NewDescriptiveError(err error, msg string, args ...interface{}) error {
	exitCode := SystemFailCode
	switch msg {
	case DetectFailMsg:
		exitCode = DetectFailCode
	case CompileFailMsg:
		exitCode = CompileFailCode
	case ReleaseFailMsg:
		exitCode = ReleaseFailCode
	case SupplyFailMsg, NoSupplyScriptFailMsg:
		exitCode = SupplyFailCode
	case FinalizeFailMsg:
		exitCode = FinalizeFailCode
	default:
		exitCode = SystemFailCode
	}

	if len(args) == 0 {
		return DescriptiveError{Message: msg, ExitCode: exitCode, InnerError: err}
	}
	return DescriptiveError{Message: fmt.Sprintf(msg, args...), InnerError: err, ExitCode: exitCode}
}
