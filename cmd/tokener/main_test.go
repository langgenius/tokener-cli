package main

import (
	"io"
	"testing"

	"github.com/spf13/cobra"
)

func TestConfigureAuthLoginHostnamePrecedence(t *testing.T) {
	tests := []struct {
		name     string
		env      string
		explicit string
		want     string
	}{
		{name: "default", want: managementHostname},
		{name: "environment", env: "env.tokener.test", want: "env.tokener.test"},
		{name: "explicit", env: "env.tokener.test", explicit: "flag.tokener.test", want: "flag.tokener.test"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(hostEnvironment, test.env)
			root := &cobra.Command{Use: "tokener"}
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			root.PersistentFlags().String("hostname", "", "")
			var got string
			login := &cobra.Command{
				Use: "login",
				RunE: func(cmd *cobra.Command, _ []string) error {
					got, _ = cmd.Root().PersistentFlags().GetString("hostname")
					return nil
				},
			}
			auth := &cobra.Command{Use: "auth"}
			auth.AddCommand(login)
			root.AddCommand(auth)
			if err := configureAuthLogin(root); err != nil {
				t.Fatal(err)
			}
			args := []string{"auth", "login"}
			if test.explicit != "" {
				args = append(args, "--hostname", test.explicit)
			}
			root.SetArgs(args)
			if err := root.Execute(); err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("hostname = %q, want %q", got, test.want)
			}
		})
	}
}
