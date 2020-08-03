package eirinistaging

import (
	"encoding/json"
	"fmt"

	"code.cloudfoundry.org/eirini-staging/builder"
	"code.cloudfoundry.org/runtimeschema/cc_messages"
)

type BuildpacksKeyModifier struct {
	CCBuildpacksJSON string
}

func (m *BuildpacksKeyModifier) Modify(result builder.StagingResult) (builder.StagingResult, error) {
	buildpacks, err := m.getProvidedBuildpacks()
	if err != nil {
		return builder.StagingResult{}, err
	}

	if err := m.modifyBuildpackKey(&result, buildpacks); err != nil {
		return builder.StagingResult{}, err
	}

	if err := m.modifyBuildpacks(&result, buildpacks); err != nil {
		return builder.StagingResult{}, err
	}

	return result, nil
}

func (m *BuildpacksKeyModifier) modifyBuildpackKey(result *builder.StagingResult, buildpacks []cc_messages.Buildpack) error {
	name := result.LifecycleMetadata.BuildpackKey
	key, err := m.getBuildpackKey(name, buildpacks)
	if err != nil {
		return err
	}
	result.LifecycleMetadata.BuildpackKey = key

	return nil
}

func (m *BuildpacksKeyModifier) modifyBuildpacks(result *builder.StagingResult, buildpacks []cc_messages.Buildpack) error {
	for i, b := range result.LifecycleMetadata.Buildpacks {
		modified, err := m.modifyBuildpackMetadata(b, buildpacks)
		if err != nil {
			return err
		}

		result.LifecycleMetadata.Buildpacks[i] = modified
	}

	return nil
}

func (m *BuildpacksKeyModifier) modifyBuildpackMetadata(b builder.BuildpackMetadata, buildpacks []cc_messages.Buildpack) (builder.BuildpackMetadata, error) {
	name := b.Key
	key, err := m.getBuildpackKey(name, buildpacks)
	if err != nil {
		return builder.BuildpackMetadata{}, err
	}
	b.Key = key

	return b, nil
}

func (m *BuildpacksKeyModifier) getProvidedBuildpacks() ([]cc_messages.Buildpack, error) {
	var providedBuildpacks []cc_messages.Buildpack
	err := json.Unmarshal([]byte(m.CCBuildpacksJSON), &providedBuildpacks)
	if err != nil {
		return []cc_messages.Buildpack{}, err
	}

	return providedBuildpacks, nil
}

func (m *BuildpacksKeyModifier) getBuildpackKey(name string, providedBuildpacks []cc_messages.Buildpack) (string, error) {
	for _, b := range providedBuildpacks {
		if b.Name == name {
			return b.Key, nil
		}
	}

	return "", fmt.Errorf("could not find buildpack with name: %s", name)
}
