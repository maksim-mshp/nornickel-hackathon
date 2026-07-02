package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Runtime struct {
	Log         Log               `koanf:"log"`
	HTTP        HTTP              `koanf:"http"`
	GRPC        GRPC              `koanf:"grpc"`
	Health      Health            `koanf:"health"`
	GRPCClients map[string]string `koanf:"grpc_clients"`
	Postgres    Postgres          `koanf:"postgres"`
	NATS        NATS              `koanf:"nats"`
}

type Postgres struct {
	DSN      string `koanf:"dsn"`
	MaxConns int32  `koanf:"max_conns"`
}

type NATS struct {
	URL     string       `koanf:"url"`
	Streams []NATSStream `koanf:"streams"`
}

type NATSStream struct {
	Name     string   `koanf:"name"`
	Subjects []string `koanf:"subjects"`
}

type Log struct {
	Level  string `koanf:"level"`
	Format string `koanf:"format"`
}

type HTTP struct {
	Addr string `koanf:"addr"`
}

type GRPC struct {
	Addr string `koanf:"addr"`
}

type Health struct {
	Addr string `koanf:"addr"`
}

type Bundle struct {
	Service string
	Env     string
	Root    string
	Runtime Runtime
	Raw     map[string]any
}

func Load(root string, env string, service string) (Bundle, error) {
	if root == "" {
		return Bundle{}, errors.New("config root is required")
	}

	if err := validateEnv(env); err != nil {
		return Bundle{}, err
	}

	store := koanf.NewWithConf(koanf.Conf{
		Delim:       ".",
		StrictMerge: true,
	})
	required := []string{
		filepath.Join(root, "base", "common.yml"),
		filepath.Join(root, "base", service+".yml"),
	}
	optional := []string{
		filepath.Join(root, env, "common.yml"),
		filepath.Join(root, env, service+".yml"),
		filepath.Join(root, "secrets.yml"),
	}

	for _, path := range required {
		if err := loadFile(store, path); err != nil {
			return Bundle{}, err
		}
	}

	for _, path := range optional {
		if err := loadOptionalFile(store, path); err != nil {
			return Bundle{}, err
		}
	}

	var runtime Runtime
	if err := store.Unmarshal("", &runtime); err != nil {
		return Bundle{}, fmt.Errorf("decode runtime config: %w", err)
	}

	if err := validateRuntime(service, runtime); err != nil {
		return Bundle{}, err
	}

	return Bundle{Service: service, Env: env, Root: root, Runtime: runtime, Raw: store.Raw()}, nil
}

func validateEnv(env string) error {
	switch env {
	case "dev", "demo", "prod":
		return nil
	default:
		return fmt.Errorf("unsupported env %q", env)
	}
}

func validateRuntime(service string, runtime Runtime) error {
	if runtime.Log.Level == "" {
		return errors.New("log.level is required")
	}
	if runtime.Log.Format == "" {
		return errors.New("log.format is required")
	}
	if service == "gateway" && runtime.HTTP.Addr == "" {
		return errors.New("http.addr is required for gateway")
	}
	if service != "gateway" && runtime.GRPC.Addr == "" {
		return fmt.Errorf("grpc.addr is required for %s", service)
	}
	if runtime.Health.Addr == "" && runtime.HTTP.Addr == "" {
		return errors.New("health.addr or http.addr is required")
	}
	return nil
}

func loadFile(store *koanf.Koanf, path string) error {
	if err := store.Load(file.Provider(path), yaml.Parser()); err != nil {
		return fmt.Errorf("load config %s: %w", path, err)
	}
	return nil
}

func loadOptionalFile(store *koanf.Koanf, path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return loadFile(store, path)
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return fmt.Errorf("stat config %s: %w", path, err)
}
