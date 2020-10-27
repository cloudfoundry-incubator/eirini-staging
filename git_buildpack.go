package eirinistaging

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
)

func GitClone(repo url.URL, destination string) error {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return fmt.Errorf("could not find `git` in path: %w", err)
	}

	branch := repo.Fragment
	repo.Fragment = ""
	gitURL := repo.String()

	err = performGitClone(gitPath,
		[]string{
			"--depth",
			"1",
			"--recursive",
			gitURL,
			destination,
		}, branch)

	if err != nil {
		os.RemoveAll(destination)

		err = performGitClone(gitPath,
			[]string{
				"--recursive",
				gitURL,
				destination,
			}, branch)

		if err != nil {
			return fmt.Errorf("failed to clone git repository at %s", gitURL)
		}
	}

	return nil
}

func performGitClone(gitPath string, args []string, branch string) error {
	args = append([]string{"clone"}, args...)

	if branch != "" {
		args = append(args, "-b", branch)
	}
	cmd := exec.Command(gitPath, args...)

	return cmd.Run()
}
