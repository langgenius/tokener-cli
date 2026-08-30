package main

import (
	"io"
	"testing"

	"github.com/spf13/cobra"
)

func TestConfigureAuthLoginDefaults(t *testing.T) {
	tests := []struct {
		name         string
		env          string
		explicitHost string
		withToken    bool
		withoutToken bool
		authType     string
		wantHost     string
		wantAuthType string
	}{
		{name: "default", wantHost: managementHostname, wantAuthType: "oauth"},
		{name: "environment", env: "env.tokener.test", wantHost: "env.tokener.test", wantAuthType: "oauth"},
		{name: "explicit host", env: "env.tokener.test", explicitHost: "flag.tokener.test", wantHost: "flag.tokener.test", wantAuthType: "oauth"},
		{name: "implicit token type", withToken: true, wantHost: managementHostname, wantAuthType: "bearer"},
		{name: "explicit token type", withToken: true, authType: "oauth", wantHost: managementHostname, wantAuthType: "oauth"},
		{name: "disabled token mode", withoutToken: true, wantHost: managementHostname, wantAuthType: "oauth"},
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
			args := []string{"auth", "login"}
			if test.explicitHost != "" {
				args = append(args, "--hostname", test.explicitHost)
			}
			if test.withToken {
				args = append(args, "--with-token")
			}
			if test.withoutToken {
				args = append(args, "--with-token=false")
			}
			if test.authType != "" {
				args = append(args, "--auth-type", test.authType)
			}
			root.SetArgs(args)
			if err := root.Execute(); err != nil {
				t.Fatal(err)
			}
			if gotHost != test.wantHost {
				t.Fatalf("hostname = %q, want %q", gotHost, test.wantHost)
			}
			if gotAuthType != test.wantAuthType {
				t.Fatalf("auth type = %q, want %q", gotAuthType, test.wantAuthType)
			}
		})
	}
}
