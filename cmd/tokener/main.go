package main

import (
	_ "embed"
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/lathe-cli/lathe/pkg/lathe"
	"github.com/spf13/cobra"

	"github.com/langgenius/tokener-cli/internal/agent"
	"github.com/langgenius/tokener-cli/internal/generated"
)

//go:embed cli.yaml
var manifestBytes []byte

const (
	managementHostname = "console.tokener.dev"
	hostEnvironment    = "TOKENER_HOST"
)

func main() {
	os.Exit(lathe.Run(lathe.RunOptions{
		Manifest: manifestBytes,
		Mount:    mount,
		Version:  moduleVersion(),
	}))
}

func moduleVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	version := strings.TrimSpace(info.Main.Version)
	if version == "" || version == "(devel)" {
		return ""
	}
	return version
}

func mount(root *cobra.Command) error {
	if err := generated.MountModules(root); err != nil {
		return err
	}
	if err := configureAuthLogin(root); err != nil {
		return err
	}
	root.AddCommand(agent.NewCommand())
	return nil
}

func configureAuthLogin(root *cobra.Command) error {
	login, _, err := root.Find([]string{"auth", "login"})
	if err != nil {
		return fmt.Errorf("find auth login command: %w", err)
	}
	if login == root || login.Name() != "login" {
		return fmt.Errorf("find auth login command: command is unavailable")
	}
	if root.PersistentFlags().Lookup("hostname") == nil {
		return fmt.Errorf("configure auth login: hostname flag is unavailable")
	}
	previous := login.PreRunE
	login.PreRunE = func(cmd *cobra.Command, args []string) error {
		flag := cmd.Root().PersistentFlags().Lookup("hostname")
		if !flag.Changed {
			hostname := strings.TrimSpace(os.Getenv(hostEnvironment))
			if hostname == "" {
				hostname = managementHostname
			}
			if err := cmd.Root().PersistentFlags().Set("hostname", hostname); err != nil {
				return fmt.Errorf("configure auth login hostname: %w", err)
			}
		}
		if previous != nil {
			return previous(cmd, args)
		}
		return nil
	}
	return nil
}
