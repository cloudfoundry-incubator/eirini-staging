package builder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

const (
	executableMask = 0111
)

type Runner struct {
	config       *Config
	depsDir      string
	contentsDir  string
	profileDir   string
	BuildpackOut io.Writer
	BuildpackErr io.Writer
}

func NewRunner(config *Config) *Runner {
	return &Runner{
		config:       config,
		BuildpackOut: os.Stdout,
		BuildpackErr: os.Stderr,
	}
}

func (runner *Runner) Run() error {
	// set up the world
	err := runner.makeDirectories()
	if err != nil {
		return errors.Wrap(err, "Failed to set up filesystem when generating droplet")
	}

	// detect, compile, release
	log.Println("Cleaning cache dir")
	err = runner.cleanCacheDir()
	if err != nil {
		return errors.Wrap(err, "unable to clean cache dir")
	}

	log.Println("Detecting buildpack")
	detectedBuildpackDir, buildpackMetadata, err := runner.supplyOrDetect()
	if err != nil {
		// detect buildpack returns custom error
		return err
	}

	if err = runner.runFinalize(detectedBuildpackDir); err != nil {
		// runFinalize returns custom error
		return err
	}

	// re-evaluate metadata after finalize in case of multi-buildpack
	if runner.config.SkipDetect {
		buildpackMetadata = runner.buildpacksMetadata(runner.config.BuildpackOrder)
	}

	log.Println("Building droplet release")
	releaseInfo, err := runner.release(detectedBuildpackDir)
	if err != nil {
		return NewReleaseFailError(errors.Wrap(err, "Failed to build droplet release"))
	}

	err = runner.writeStagingInfoYML(releaseInfo.DefaultProcessTypes["web"], buildpackMetadata)
	if err != nil {
		return errors.Wrap(err, "unable to build staging info for the droplet")
	}

	tarPath, err := runner.findTar()
	if err != nil {
		return err
	}

	log.Println("Creating app artifact")
	err = runner.createArtifacts(tarPath, buildpackMetadata, releaseInfo)
	if err != nil {
		return errors.Wrap(err, "failed to find runnable app artifact")
	}

	err = runner.createCache(tarPath)
	if err != nil {
		return errors.Wrap(err, "failed to cache runnable app artifact")
	}

	return nil
}

func (runner *Runner) CleanUp() {
	if runner.contentsDir == "" {
		return
	}
	os.RemoveAll(runner.contentsDir)
}

func (runner *Runner) supplyOrDetect() (string, []BuildpackMetadata, error) {
	if runner.config.SkipDetect {
		return runner.runSupplyBuildpacks()
	}

	return runner.detect()
}

func (runner *Runner) createArtifacts(tarPath string, buildpackMetadata []BuildpackMetadata, releaseInfo Release) error {
	err := runner.saveInfo(buildpackMetadata, releaseInfo)
	if err != nil {
		return errors.Wrap(err, "Failed to encode generated metadata")
	}

	for _, name := range []string{"tmp", "logs"} {
		if err = os.MkdirAll(filepath.Join(runner.contentsDir, name), 0755); err != nil {
			return errors.Wrap(err, "Failed to set up droplet filesystem")
		}
	}

	appDir := filepath.Join(runner.contentsDir, "app")
	err = runner.copyApp(runner.config.BuildDir, appDir)
	if err != nil {
		return errors.Wrap(err, "Failed to copy compiled droplet")
	}

	err = exec.Command(tarPath, "-czf", runner.config.OutputDropletLocation, "-C", runner.contentsDir, ".").Run()
	if err != nil {
		return errors.Wrap(err, "Failed to compress droplet filesystem")
	}

	return nil
}

func (runner *Runner) createCache(tarPath string) error {
	err := os.MkdirAll(filepath.Dir(runner.config.OutputBuildArtifactsCache), 0755)
	if err != nil {
		return errors.Wrap(err, "Failed to create output build artifacts cache dir")
	}

	err = exec.Command(tarPath, "-czf", runner.config.OutputBuildArtifactsCache, "-C", runner.config.BuildArtifactsCacheDir(), ".").Run()

	return errors.Wrap(err, "Failed to compress build artifacts")
}

func (runner *Runner) buildpacksMetadata(buildpacks []string) []BuildpackMetadata {
	data := make([]BuildpackMetadata, len(buildpacks))
	for i, key := range buildpacks {
		data[i].Key = key
		configPath := filepath.Join(runner.depsDir, runner.config.DepsIndex(i), "config.yml")
		if contents, err := ioutil.ReadFile(configPath); err == nil {
			configyaml := struct {
				Name    string `yaml:"name"`
				Version string `yaml:"version"`
			}{}
			if err := yaml.Unmarshal(contents, &configyaml); err == nil {
				data[i].Name = configyaml.Name
				data[i].Version = configyaml.Version
			}
		}
	}

	return data
}

func (runner *Runner) makeDirectories() error {
	if err := os.MkdirAll(filepath.Dir(runner.config.OutputDropletLocation), 0755); err != nil {
		return fmt.Errorf("failed to create output droplet location directory: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(runner.config.OutputMetadataLocation), 0755); err != nil {
		return fmt.Errorf("failed to create output metadata location directory: %w", err)
	}

	if err := os.MkdirAll(filepath.Join(runner.config.BuildArtifactsCacheDir(), "final"), 0755); err != nil {
		return fmt.Errorf("failed to create build artifacts cache directory: %w", err)
	}

	for _, buildpack := range runner.config.SupplyBuildpacks() {
		if err := os.MkdirAll(runner.supplyCachePath(buildpack), 0755); err != nil {
			return fmt.Errorf("failed to create supply cache directory: %w", err)
		}
	}

	var err error
	runner.contentsDir, err = ioutil.TempDir("", "contents")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}

	runner.depsDir = filepath.Join(runner.contentsDir, "deps")

	for i := 0; i <= len(runner.config.SupplyBuildpacks()); i++ {
		if err := os.MkdirAll(filepath.Join(runner.depsDir, runner.config.DepsIndex(i)), 0755); err != nil {
			return fmt.Errorf("failed to create deps dir: %w", err)
		}
	}

	runner.profileDir = filepath.Join(runner.contentsDir, "profile.d")
	if err := os.MkdirAll(runner.profileDir, 0755); err != nil {
		return fmt.Errorf("failed to create profile dir: %w", err)
	}

	return nil
}

func (runner *Runner) cleanCacheDir() error {
	neededCacheDirs := map[string]bool{
		filepath.Join(runner.config.BuildArtifactsCacheDir(), "final"): true,
	}

	for _, bp := range runner.config.SupplyBuildpacks() {
		neededCacheDirs[runner.supplyCachePath(bp)] = true
	}

	dirs, err := ioutil.ReadDir(runner.config.BuildArtifactsCacheDir())
	if err != nil {
		return fmt.Errorf("failed to create artifacts cache dir: %w", err)
	}

	for _, dirInfo := range dirs {
		dir := filepath.Join(runner.config.BuildArtifactsCacheDir(), dirInfo.Name())
		if !neededCacheDirs[dir] {
			err = os.RemoveAll(dir)
			if err != nil {
				return fmt.Errorf("failed to remove build artifacts cache dir: %w", err)
			}
		}
	}

	return nil
}

func (runner *Runner) buildpackPath(buildpack string) (string, error) {
	buildpackPath := BuildpackPath(runner.config.BuildpacksDir, buildpack)

	if runner.pathHasBinDirectory(buildpackPath) {
		return buildpackPath, nil
	}

	files, err := ioutil.ReadDir(buildpackPath)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to read buildpack directory '%s' for buildpack '%s'", buildpackPath, buildpack)
	}

	if len(files) == 1 {
		nestedPath := filepath.Join(buildpackPath, files[0].Name())

		if runner.pathHasBinDirectory(nestedPath) {
			return nestedPath, nil
		}
	}

	return "", errors.Errorf("malformed buildpack does not contain a /bin dir: %s", buildpack)
}

func (runner *Runner) pathHasBinDirectory(pathToTest string) bool {
	_, err := os.Stat(filepath.Join(pathToTest, "bin"))

	return err == nil
}

func (runner *Runner) supplyCachePath(buildpack string) string {
	return BuildpackPath(runner.config.BuildArtifactsCacheDir(), buildpack)
}

func fileExists(file string) (bool, error) {
	_, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, fmt.Errorf("error stat'ing file: %w", err)
	}

	return true, nil
}

func (runner *Runner) runSupplyBuildpacks() (string, []BuildpackMetadata, error) {
	if err := runner.validateSupplyBuildpacks(); err != nil {
		return "", nil, err
	}

	for i, buildpack := range runner.config.SupplyBuildpacks() {
		buildpackPath, err := runner.buildpackPath(buildpack)
		if err != nil {
			return "", nil, NewSupplyFailError(err)
		}

		err = runner.run(exec.Command(filepath.Join(buildpackPath, "bin", "supply"), runner.config.BuildDir, runner.supplyCachePath(buildpack), runner.depsDir, runner.config.DepsIndex(i)))
		if err != nil {
			logError(fmt.Sprintf("supply script failed %s", err.Error()))

			return "", nil, NewSupplyFailError(err)
		}
	}

	finalBuildpack := runner.config.BuildpackOrder[len(runner.config.SupplyBuildpacks())]
	finalPath, err := runner.buildpackPath(finalBuildpack)
	if err != nil {
		return "", nil, NewSupplyFailError(err)
	}

	buildpacks := runner.buildpacksMetadata(runner.config.BuildpackOrder)

	return finalPath, buildpacks, nil
}

func (runner *Runner) validateSupplyBuildpacks() error {
	for _, buildpack := range runner.config.SupplyBuildpacks() {
		buildpackPath, err := runner.buildpackPath(buildpack)
		if err != nil {
			logError(err.Error())

			return NewSupplyFailError(err)
		}

		if hasSupply, err := hasSupply(buildpackPath); err != nil {
			logError(fmt.Sprintf("failed to check if supply script exists %s", err.Error()))

			return NewSupplyFailError(err)
		} else if !hasSupply {
			logError("supply script missing")

			return NewNoSupplyScriptFailError(err)
		}
	}

	return nil
}

func (runner *Runner) runFinalize(buildpackPath string) error {
	depsIdx := runner.config.DepsIndex(len(runner.config.SupplyBuildpacks()))
	cacheDir := filepath.Join(runner.config.BuildArtifactsCacheDir(), "final")

	hasFinalize, err := hasFinalize(buildpackPath)
	if err != nil {
		return NewFinalizeFailError(err)
	}

	if hasFinalize {
		if err = runner.handleFinalize(buildpackPath, cacheDir, depsIdx); err != nil {
			return err
		}
	} else {
		if err = runner.handleNoFinalize(buildpackPath, cacheDir, depsIdx); err != nil {
			return err
		}
	}

	return nil
}

func (runner *Runner) handleFinalize(buildpackPath, cacheDir, depsIdx string) error {
	if err := runner.handleSupply(buildpackPath, cacheDir, depsIdx); err != nil {
		return err
	}

	if err := runner.run(exec.Command(filepath.Join(buildpackPath, "bin", "finalize"), runner.config.BuildDir, cacheDir, runner.depsDir, depsIdx, runner.profileDir)); err != nil {
		return NewFinalizeFailError(err)
	}

	return nil
}

func (runner *Runner) handleNoFinalize(buildpackPath, cacheDir, depsIdx string) error {
	if len(runner.config.SupplyBuildpacks()) > 0 {
		logError(MissingFinalizeWarnMsg)
	}

	// remove unused deps sub dir
	if err := os.RemoveAll(filepath.Join(runner.depsDir, depsIdx)); err != nil {
		return NewCompileFailError(err)
	}

	if err := runner.run(exec.Command(filepath.Join(buildpackPath, "bin", "compile"), runner.config.BuildDir, cacheDir)); err != nil {
		logError(fmt.Sprintf("compile script failed %s", err.Error()))

		return NewCompileFailError(errors.Wrap(err, "failed to compile droplet"))
	}

	return nil
}

func (runner *Runner) handleSupply(buildpackPath, cacheDir, depsIdx string) error {
	hasSupply, err := hasSupply(buildpackPath)
	if err != nil {
		return NewSupplyFailError(err)
	}

	if hasSupply {
		if err := runner.run(exec.Command(filepath.Join(buildpackPath, "bin", "supply"), runner.config.BuildDir, cacheDir, runner.depsDir, depsIdx)); err != nil {
			return NewSupplyFailError(err)
		}
	}

	return nil
}

func (runner *Runner) detect() (string, []BuildpackMetadata, error) {
	for _, buildpack := range runner.config.BuildpackOrder {
		buildpackPath, err := runner.buildpackPath(buildpack)
		if err != nil {
			logError(err.Error())

			continue
		}

		if err = runner.warnIfDetectNotExecutable(buildpackPath); err != nil {
			logError(err.Error())

			continue
		}

		output, err := runner.runWithCapturing(exec.Command(filepath.Join(buildpackPath, "bin", "detect"), runner.config.BuildDir))

		if err == nil {
			buildpacks := runner.buildpacksMetadata([]string{buildpack})
			if buildpacks[0].Name == "" {
				buildpacks[0].Name = strings.TrimRight(output.String(), "\r\n")
			}

			return buildpackPath, buildpacks, nil
		}
	}

	return "", nil, DescriptiveError{ExitCode: DetectFailCode, Message: DetectFailMsg, InnerError: errors.New(FullDetectFailMsg)}
}

func (runner *Runner) readProcfile() (map[string]string, error) {
	processes := map[string]string{}

	procFile, err := ioutil.ReadFile(filepath.Join(runner.config.BuildDir, "Procfile"))
	if err != nil {
		if os.IsNotExist(err) {
			// Procfiles are optional
			return processes, nil
		}

		return processes, fmt.Errorf("error reading Procfile: %w", err)
	}

	err = yaml.Unmarshal(procFile, &processes)
	if err != nil {
		// clobber yaml parsing  error
		return processes, errors.Wrap(err, "invalid YAML")
	}

	return processes, nil
}

func (runner *Runner) release(buildpackDir string) (Release, error) {
	startCommands, err := runner.readProcfile()
	if err != nil {
		return Release{}, errors.Wrap(err, "Failed to read command from Procfile")
	}

	output, err := runner.runWithCapturing(exec.Command(filepath.Join(buildpackDir, "bin", "release"), runner.config.BuildDir))
	if err != nil {
		return Release{}, errors.Wrap(err, "no release script")
	}

	parsedRelease := Release{}

	err = yaml.Unmarshal(output.Bytes(), &parsedRelease)
	if err != nil {
		return Release{}, errors.Wrap(err, "buildpack's release output invalid")
	}

	if len(startCommands) > 0 {
		if len(parsedRelease.DefaultProcessTypes) == 0 {
			parsedRelease.DefaultProcessTypes = startCommands
		} else {
			for k, v := range startCommands {
				parsedRelease.DefaultProcessTypes[k] = v
			}
		}
	}

	if parsedRelease.DefaultProcessTypes["web"] == "" {
		logError("No start command specified by buildpack or via Procfile.")
		logError("App will not start unless a command is provided at runtime.")
	}

	return parsedRelease, nil
}

func (runner *Runner) saveInfo(buildpacks []BuildpackMetadata, releaseInfo Release) error {
	var lastBuildpack BuildpackMetadata
	if len(buildpacks) > 0 {
		lastBuildpack = buildpacks[len(buildpacks)-1]
	}

	resultFile, err := os.Create(runner.config.OutputMetadataLocation)
	if err != nil {
		return fmt.Errorf("failed to create output metadata location: %w", err)
	}
	defer resultFile.Close()

	return json.NewEncoder(resultFile).Encode(NewStagingResult(
		releaseInfo.DefaultProcessTypes,
		LifecycleMetadata{
			BuildpackKey:      lastBuildpack.Key,
			DetectedBuildpack: lastBuildpack.Name,
			Buildpacks:        buildpacks,
		},
	))
}

func (runner *Runner) run(cmd *exec.Cmd) error {
	cmd.Stdout = runner.BuildpackOut
	cmd.Stderr = runner.BuildpackErr

	return cmd.Run()
}

func (runner *Runner) runWithCapturing(cmd *exec.Cmd) (*bytes.Buffer, error) {
	output := new(bytes.Buffer)
	cmd.Stdout = output
	cmd.Stderr = runner.BuildpackErr

	return output, cmd.Run()
}

func (runner *Runner) copyApp(buildDir, stageDir string) error {
	return runner.run(exec.Command("cp", "-a", buildDir, stageDir))
}

func (runner *Runner) warnIfDetectNotExecutable(buildpackPath string) error {
	fileInfo, err := os.Stat(filepath.Join(buildpackPath, "bin", "detect"))
	if err != nil {
		return errors.Wrap(err, "failed to find detect script")
	}

	if fileInfo.Mode()&executableMask != executableMask {
		log.Println("WARNING: buildpack script '/bin/detect' is not executable")
	}

	return nil
}

func (runner *Runner) findTar() (string, error) {
	tarPath, err := exec.LookPath("tar")
	if err != nil {
		return "", fmt.Errorf("failed to find `tar` in the path: %w", err)
	}

	return tarPath, nil
}

func (runner *Runner) writeStagingInfoYML(startCommand string, buildpacks []BuildpackMetadata) error {
	stagingInfoYML := filepath.Join(runner.contentsDir, "staging_info.yml")
	stagingInfoFile, err := os.Create(stagingInfoYML)
	if err != nil {
		return fmt.Errorf("failed to create	staging_info.yml: %w", err)
	}
	defer stagingInfoFile.Close()

	// var lastBuildpack buildpackapplifecycle.BuildpackMetadata
	// if len(buildpacks) > 0 {
	lastBuildpack := buildpacks[len(buildpacks)-1]
	// }

	return json.NewEncoder(stagingInfoFile).Encode(StagingInfo{
		DetectedBuildpack: lastBuildpack.Name,
		StartCommand:      startCommand,
	})
}
