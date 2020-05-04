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

func (s *Config) InitBuildpacks(buildpacksJSON string) error {
	if buildpacksJSON != "" {
		var buildpacks []Buildpack
		err := json.Unmarshal([]byte(buildpacksJSON), &buildpacks)
		if err != nil {
			return err
		}

		if len(buildpacks) > 0 {
			s.SkipDetect = true
			for _, b := range buildpacks {
				s.BuildpackOrder = append(s.BuildpackOrder, b.Name)
				s.SkipDetect = s.SkipDetect && b.SkipDetect
			}
		}
	}

	return nil
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
	numBuildpacks := len(s.SupplyBuildpacks()) + 1
	padDigits := int(math.Log10(float64(numBuildpacks))) + 1
	indexFormat := fmt.Sprintf("%%0%dd", padDigits)
	return fmt.Sprintf(indexFormat, i)
}
