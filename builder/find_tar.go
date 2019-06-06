package builder

import "os/exec"

func (runner *Runner) findTar() (string, error) {
	tarPath, err := exec.LookPath("tar")
	if err != nil {
		return "", err
	}
	return tarPath, nil
}
