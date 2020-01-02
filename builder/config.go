package builder

import (
	"encoding/json"
	"fmt"
	"math"
)

type Config struct {
	BuildDir                  string
	BuildpacksDir             string
	OutputDropletLocation     string
	OutputBuildArtifactsCache string
	OutputMetadataLocation    string
	BuildpackOrder            []string
	SkipDetect                bool
	BuildArtifactsCache       string
}

func NewConfig(
	buildDir string,
	buildpacksDir string,
	outputDropletLocation string,
	outputBuildArtifactsCache string,
	outputMetadataLocation string,
	buildArtifactsCache string,
	buildpackJSON string,
) (Config, error) {
	cfg := Config{
		BuildDir:                  buildDir,
		BuildpacksDir:             buildpacksDir,
		OutputDropletLocation:     outputDropletLocation,
		OutputBuildArtifactsCache: outputBuildArtifactsCache,
		OutputMetadataLocation:    outputMetadataLocation,
		BuildArtifactsCache:       buildArtifactsCache,
	}
	return cfg.init(buildpackJSON)
}

func (s Config) init(buildpacksJSON string) (Config, error) {
	if buildpacksJSON != "" {
		var buildpacks []Buildpack
		err := json.Unmarshal([]byte(buildpacksJSON), &buildpacks)
		if err != nil {
			return Config{}, err
		}

		if len(buildpacks) == 1 && buildpacks[0].SkipDetect {
			s.SkipDetect = true
			s.BuildpackOrder = []string{buildpacks[0].Name}
		} else if len(buildpacks) > 0 {
			for _, b := range buildpacks {
				s.BuildpackOrder = append(s.BuildpackOrder, b.Name)
			}
		}
	}

	return s, nil
}

func (s Config) BuildArtifactsCacheDir() string {
	return s.BuildArtifactsCache
}

func (s Config) SupplyBuildpacks() []string {
	numBuildpacks := len(s.BuildpackOrder)
	if !s.SkipDetect || numBuildpacks == 0 {
		return []string{}
	}
	return s.BuildpackOrder[0 : numBuildpacks-1]
}

func (s Config) DepsIndex(i int) string {
	numBuildpacks := len(s.SupplyBuildpacks()) + 1           // nolint:gomnd
	padDigits := int(math.Log10(float64(numBuildpacks))) + 1 // nolint:gomnd
	indexFormat := fmt.Sprintf("%%0%dd", padDigits)
	return fmt.Sprintf(indexFormat, i)
}
