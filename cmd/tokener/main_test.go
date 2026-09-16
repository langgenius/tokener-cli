package main

import (
	"io"
	"testing"

	"github.com/spf13/cobra"
)

func TestConfigureAuthLoginDefaults(t *testing.T) {
	tests := []struct {
		name, env  string
		args       []string
		host, auth string
	}{
		{"default", "", nil, managementHostname, "oauth"},
		{"environment", "env.tokener.test", nil, "env.tokener.test", "oauth"},
		{"explicit host", "env.tokener.test", []string{"--hostname", "flag.tokener.test"}, "flag.tokener.test", "oauth"},
		{"implicit token type", "", []string{"--with-token"}, managementHostname, "bearer"},
		{"explicit token type", "", []string{"--with-token", "--auth-type", "oauth"}, managementHostname, "oauth"},
		{"disabled token mode", "", []string{"--with-token=false"}, managementHostname, "oauth"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(hostEnvironment, test.env)
			root := &cobra.Command{Use: "tokener"}
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			root.PersistentFlags().String("hostname", "", "")
			var gotHost, gotAuthType string
			login := &cobra.Command{
				Use: "login",
				RunE: func(cmd *cobra.Command, _ []string) error {
					gotHost, _ = cmd.Root().PersistentFlags().GetString("hostname")
					gotAuthType, _ = cmd.Flags().GetString("auth-type")
					return nil
				},
			}
			login.Flags().String("auth-type", "oauth", "")
			login.Flags().Bool("with-token", false, "")
			auth := &cobra.Command{Use: "auth"}
			auth.AddCommand(login)
			root.AddCommand(auth)
			if err := configureAuthLogin(root); err != nil {
				t.Fatal(err)
			}
			root.SetArgs(append([]string{"auth", "login"}, test.args...))
			if err := root.Execute(); err != nil {
				t.Fatal(err)
			}
			if gotHost != test.host {
				t.Fatalf("hostname = %q, want %q", gotHost, test.host)
			}
			if gotAuthType != test.auth {
				t.Fatalf("auth type = %q, want %q", gotAuthType, test.auth)
			}
		})
	}
}
