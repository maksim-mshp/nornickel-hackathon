package config

import "testing"

func TestLoadMergesBaseAndEnvConfigs(t *testing.T) {
	t.Parallel()

	bundle, err := Load("../../../configs", "dev", "search")
	if err != nil {
		t.Fatalf("expected config to load: %v", err)
	}

	if bundle.Runtime.Log.Level != "debug" {
		t.Fatalf("expected dev log level, got %q", bundle.Runtime.Log.Level)
	}

	if bundle.Runtime.Log.Format != "text" {
		t.Fatalf("expected dev log format, got %q", bundle.Runtime.Log.Format)
	}

	if bundle.Runtime.GRPC.Addr != ":9093" {
		t.Fatalf("expected base grpc addr, got %q", bundle.Runtime.GRPC.Addr)
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
