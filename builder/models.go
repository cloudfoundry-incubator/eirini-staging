package builder

type Release struct {
	DefaultProcessTypes ProcessTypes `yaml:"default_process_types"`
}

// StagingInfo is used for export/import droplets.
type StagingInfo struct {
	DetectedBuildpack string `json:"detected_buildpack" yaml:"detected_buildpack"`
	StartCommand      string `json:"start_command" yaml:"start_command"`
}

type ProcessTypes map[string]string

type Buildpack struct {
	Name       string `json:"name"`
	Key        string `json:"key"`
	URL        string `json:"url"`
	SkipDetect bool   `json:"skip_detect,omitempty"`
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
