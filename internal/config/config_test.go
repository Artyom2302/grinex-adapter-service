package config

import "testing"

func TestLoad_Priority(t *testing.T) {
	//nolint:gosec // local docker default, not a secret.
	const def = "postgres://rates:rates@localhost:5432/rates?sslmode=disable"

	tests := []struct {
		name    string
		env     string
		args    []string
		wantDSN string
	}{
		{name: "default", wantDSN: def},
		{name: "env only", env: "env-dsn", wantDSN: "env-dsn"},
		{name: "flag beats env", env: "env-dsn", args: []string{"-dsn=flag-dsn"}, wantDSN: "flag-dsn"},
		{name: "flag only", args: []string{"-dsn=flag-dsn"}, wantDSN: "flag-dsn"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.env != "" {
				t.Setenv("DATABASE_DSN", tt.env)
			}
			cfg, err := Load(tt.args)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if cfg.DSN != tt.wantDSN {
				t.Errorf("DSN = %q, want %q", cfg.DSN, tt.wantDSN)
			}
		})
	}
}

func TestLoad_UnknownFlag(t *testing.T) {
	_, err := Load([]string{"-none"})
	if err == nil {
		t.Errorf("Load: got %v, want error", err)
	}
}
