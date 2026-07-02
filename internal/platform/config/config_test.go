package config

import "testing"

func TestMergeOverridesNestedValues(t *testing.T) {
	t.Parallel()

	dst := map[string]any{
		"log": map[string]any{
			"level":  "info",
			"format": "json",
		},
		"postgres": map[string]any{
			"dsn":       "base",
			"max_conns": 10,
		},
	}
	src := map[string]any{
		"log": map[string]any{
			"level": "debug",
		},
		"postgres": map[string]any{
			"dsn": "dev",
		},
	}

	merge(dst, src)

	logValues := dst["log"].(map[string]any)
	if logValues["level"] != "debug" {
		t.Fatalf("expected debug level, got %v", logValues["level"])
	}
	if logValues["format"] != "json" {
		t.Fatalf("expected inherited format, got %v", logValues["format"])
	}

	postgresValues := dst["postgres"].(map[string]any)
	if postgresValues["dsn"] != "dev" {
		t.Fatalf("expected dev dsn, got %v", postgresValues["dsn"])
	}
	if postgresValues["max_conns"] != 10 {
		t.Fatalf("expected inherited max_conns, got %v", postgresValues["max_conns"])
	}
}

func TestValidateRuntimeRequiresGatewayHTTPAddr(t *testing.T) {
	t.Parallel()

	err := validateRuntime("gateway", Runtime{
		Log:    Log{Level: "info", Format: "json"},
		Health: Health{Addr: ":8081"},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateRuntimeAcceptsGRPCServiceWithHealthAddr(t *testing.T) {
	t.Parallel()

	err := validateRuntime("search", Runtime{
		Log:    Log{Level: "info", Format: "json"},
		GRPC:   GRPC{Addr: ":9093"},
		Health: Health{Addr: ":8093"},
	})
	if err != nil {
		t.Fatalf("expected valid runtime config: %v", err)
	}
}
