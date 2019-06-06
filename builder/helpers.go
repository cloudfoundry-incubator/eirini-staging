package builder

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func hasFinalize(buildpackPath string) (bool, error) {
	return fileExists(filepath.Join(buildpackPath, "bin", "finalize"))
}

func hasSupply(buildpackPath string) (bool, error) {
	return fileExists(filepath.Join(buildpackPath, "bin", "supply"))
}

func (runner *Runner) copyApp(buildDir, stageDir string) error {
	return runner.run(exec.Command("cp", "-a", buildDir, stageDir), os.Stdout)
}

func (runner *Runner) warnIfDetectNotExecutable(buildpackPath string) error {
	fileInfo, err := os.Stat(filepath.Join(buildpackPath, "bin", "detect"))
	if err != nil {
		return err
	}

	if fileInfo.Mode()&0111 != 0111 {
		fmt.Println("WARNING: buildpack script '/bin/detect' is not executable")
	}

	return nil
}

// StringifyBuildpack converts a buildpack's fields to all strings: the format expected by the buildpackapplifecycle.
// func StringifyBuildpack(buildpack builder.Buildpack) StringifiedBuildpack {
//
// 	skipDetect := ""
// 	if buildpack.SkipDetect != nil {
// 		skipDetect = strconv.FormatBool(*buildpack.SkipDetect)
// 	}
//
// 	return StringifiedBuildpack{
// 		Name:       buildpack.Name,
// 		Key:        buildpack.Key,
// 		URL:        buildpack.URL,
// 		SkipDetect: skipDetect,
// 	}
// }
//
// // UnStringifyBuildpack converts a stringifyBuildpack back to its original self.
// func UnStringifyBuildpack(buildpack StringifiedBuildpack) (*builder.Buildpack, error) {
//
// 	var skipDetect *bool
//
// 	if buildpack.SkipDetect != "" {
// 		detect, err := strconv.ParseBool(buildpack.SkipDetect)
// 		if err != nil {
// 			return nil, err
// 		}
//
// 		skipDetect = &detect
// 	}
//
// 	return &builder.Buildpack{
// 		Name:       buildpack.Name,
// 		Key:        buildpack.Key,
// 		URL:        buildpack.URL,
// 		SkipDetect: skipDetect,
// 	}, nil
// }
