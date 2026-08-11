package cmd

import "testing"

func TestNewMCPServer_InsecureSkipTLSVerifyFlag(t *testing.T) {
	cmd := NewMCPServer(IOStreams{})
	flag := cmd.Flags().Lookup("insecure-skip-tls-verify")
	if flag == nil {
		t.Fatal("expected insecure-skip-tls-verify flag")
	}
	if flag.DefValue != "false" {
		t.Errorf("insecure-skip-tls-verify default = %q, want false", flag.DefValue)
	}
}
