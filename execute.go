package eirinistaging

import (
	"os"

	"code.cloudfoundry.org/eirini-staging/builder"
)

type IOCommander struct {
	Stdout   *os.File
	Stderr   *os.File
	Stdin    *os.File
	ExitCode int
}

type PacksExecutor struct {
	Conf *builder.Config
}

func (e *PacksExecutor) ExecuteRecipe() error {
	runner := builder.NewRunner(e.Conf)
	defer runner.CleanUp()

	err := runner.Run()
	if err != nil {
		return err
	}

	return nil
}
