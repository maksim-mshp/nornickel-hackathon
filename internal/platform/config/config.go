package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Runtime struct {
	Log    Log    `yaml:"log"`
	HTTP   HTTP   `yaml:"http"`
	GRPC   GRPC   `yaml:"grpc"`
	Health Health `yaml:"health"`
}

type Log struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

type HTTP struct {
	Addr string `yaml:"addr"`
}

type GRPC struct {
	Addr string `yaml:"addr"`
}

type Health struct {
	Addr string `yaml:"addr"`
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

	merged := map[string]any{}
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
		next, err := readMap(path)
		if err != nil {
			return Bundle{}, err
		}
		merge(merged, next)
	}

	for _, path := range optional {
		next, err := readOptionalMap(path)
		if err != nil {
			return Bundle{}, err
		}
		merge(merged, next)
	}

	runtime, err := decodeRuntime(merged)
	if err != nil {
		return Bundle{}, err
	}

	if err := validateRuntime(service, runtime); err != nil {
		return Bundle{}, err
	}

	return Bundle{Service: service, Env: env, Root: root, Runtime: runtime, Raw: merged}, nil
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

func decodeRuntime(values map[string]any) (Runtime, error) {
	data, err := yaml.Marshal(values)
	if err != nil {
		return Runtime{}, fmt.Errorf("encode merged config: %w", err)
	}

	var runtime Runtime
	if err := yaml.Unmarshal(data, &runtime); err != nil {
		return Runtime{}, fmt.Errorf("decode runtime config: %w", err)
	}

	return runtime, nil
}

func readMap(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	values := map[string]any{}
	if err := yaml.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	return values, nil
}

func readOptionalMap(path string) (map[string]any, error) {
	_, err := os.Stat(path)
	if err == nil {
		return readMap(path)
	}
	if errors.Is(err, os.ErrNotExist) {
		return map[string]any{}, nil
	}
	return nil, fmt.Errorf("stat config %s: %w", path, err)
}

func merge(dst map[string]any, src map[string]any) {
	for key, srcValue := range src {
		dstMap, dstOK := dst[key].(map[string]any)
		srcMap, srcOK := srcValue.(map[string]any)
		if dstOK && srcOK {
			merge(dstMap, srcMap)
			continue
		}
		dst[key] = srcValue
	}
}
