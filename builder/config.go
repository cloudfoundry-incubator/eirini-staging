package builder

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
)

type Config struct {
	BuildDir                  string
	BuildpacksDir             string
	OutputDropletLocation     string
	OutputBuildArtifactsCache string
	OutputMetadataLocation    string
	BuildpackOrder            []string
	SkipDetect                bool
}

func (s Config) Init(buildpacksJSON string) (Config, error) {
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
	return "/tmp/cache"
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

func (s Config) BuildpackPath(buildpackName string) string {
	baseDir := s.BuildpacksDir
	// buildpackURL, err := url.Parse(buildpackName)
	// if err == nil && buildpackURL.IsAbs() {
	// 	baseDir = s.BuildpacksDownloadDir()
	// }
	return filepath.Join(baseDir, fmt.Sprintf("%x", md5.Sum([]byte(buildpackName))))
}
