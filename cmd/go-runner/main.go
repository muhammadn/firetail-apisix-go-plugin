package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"


        _ "firetail_apisix_plugin/cmd/go-runner/plugins"
	"firetail_apisix_plugin/pkg/runner"
	"firetail_apisix_plugin/pkg/log"
)

const (
	LogFilePath     = "./logs/runner.log"
)

func openFileToWrite(name string) (*os.File, error) {
	dir := filepath.Dir(name)
	if dir != "." {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return nil, err
		}
	}
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func newRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "run",
		Run: func(cmd *cobra.Command, _ []string) {
			cfg := runner.RunnerConfig{}
			f, err := openFileToWrite(LogFilePath)
			if err != nil {
				log.Fatalf("failed to open log: %s", err)
			}
			cfg.LogOutput = f

			runner.Run(cfg)
		},
	}
	return cmd
}

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "go-runner [command]",
		Long:    "The Plugin runner to run Firetail Go plugins",
		Version: "0.0.1",
	}

	cmd.AddCommand(newRunCommand())
	return cmd
}

func main() {
	root := NewCommand()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
