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
	S3          S3                `koanf:"s3"`
	LLM         LLM               `koanf:"llm"`
	Cache       Cache             `koanf:"cache"`
	Budget      Budget            `koanf:"budget"`
	Auth        Auth              `koanf:"auth"`
	Bootstrap   Bootstrap         `koanf:"bootstrap"`
}

type Budget struct {
	FirstTokenMS int `koanf:"first_token_ms"`
	TotalMS      int `koanf:"total_ms"`
	SynthesisMS  int `koanf:"synthesis_ms"`
}

type Bootstrap struct {
	CorpusDir         string   `koanf:"corpus_dir"`
	IncludeExtensions []string `koanf:"include_extensions"`
	Geography         string   `koanf:"geography"`
	AccessLevel       string   `koanf:"access_level"`
	Concurrency       int      `koanf:"concurrency"`
	MaxFileMB         int      `koanf:"max_file_mb"`
}

type Auth struct {
	Mode       string   `koanf:"mode"`
	SigningKey string   `koanf:"signing_key"`
	Demo       AuthDemo `koanf:"demo"`
	OIDC       AuthOIDC `koanf:"oidc"`
}

type AuthDemo struct {
	Tokens map[string]AuthDemoToken `koanf:"tokens"`
}

type AuthDemoToken struct {
	Sub       string   `koanf:"sub"`
	Name      string   `koanf:"name"`
	Roles     []string `koanf:"roles"`
	DocAccess string   `koanf:"doc_access"`
}

type AuthOIDC struct {
	Issuer         string `koanf:"issuer"`
	Audience       string `koanf:"audience"`
	DocAccessClaim string `koanf:"doc_access_claim"`
}

type LLM struct {
	DefaultProvider   string                 `koanf:"default_provider"`
	FallbackProviders []string               `koanf:"fallback_providers"`
	LogPrompts        bool                   `koanf:"log_prompts"`
	Allowlist         []string               `koanf:"allowlist"`
	Providers         map[string]LLMProvider `koanf:"providers"`
	Tasks             map[string]LLMTask     `koanf:"tasks"`
	Concurrency       LLMConcurrency         `koanf:"concurrency"`
}

type LLMProvider struct {
	BaseURL    string            `koanf:"base_url"`
	APIKey     string            `koanf:"api_key"`
	AuthScheme string            `koanf:"auth_scheme"`
	FolderID   string            `koanf:"folder_id"`
	Models     map[string]string `koanf:"models"`
}

type LLMTask struct {
	Model           string  `koanf:"model"`
	FallbackModel   string  `koanf:"fallback_model"`
	EscalateModel   string  `koanf:"escalate_model"`
	MaxTokens       int     `koanf:"max_tokens"`
	Temperature     float64 `koanf:"temperature"`
	ReasoningEffort string  `koanf:"reasoning_effort"`
	JSON            bool    `koanf:"json"`
	Stream          bool    `koanf:"stream"`
	TimeoutS        int     `koanf:"timeout_s"`
}

type LLMConcurrency struct {
	Interactive int `koanf:"interactive"`
	Batch       int `koanf:"batch"`
}

type Cache struct {
	TTLHours int `koanf:"ttl_hours"`
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

type S3 struct {
	Endpoint  string            `koanf:"endpoint"`
	AccessKey string            `koanf:"access_key"`
	SecretKey string            `koanf:"secret_key"`
	UseSSL    bool              `koanf:"use_ssl"`
	Region    string            `koanf:"region"`
	Buckets   map[string]string `koanf:"buckets"`
}

type Log struct {
	Level  string `koanf:"level"`
	Format string `koanf:"format"`
}

type HTTP struct {
	Addr        string   `koanf:"addr"`
	CorsOrigins []string `koanf:"cors_origins"`
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
		filepath.Join(root, "base", service+"-routes.yml"),
		filepath.Join(root, env, "common.yml"),
		filepath.Join(root, env, service+".yml"),
		filepath.Join(root, env, service+"-routes.yml"),
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

func LoadNamed(root string, env string, name string, out any) error {
	if root == "" {
		return errors.New("config root is required")
	}
	if err := validateEnv(env); err != nil {
		return err
	}

	store := koanf.NewWithConf(koanf.Conf{Delim: ".", StrictMerge: true})
	if err := loadFile(store, filepath.Join(root, "base", name+".yml")); err != nil {
		return err
	}
	for _, path := range []string{
		filepath.Join(root, env, name+".yml"),
		filepath.Join(root, "secrets.yml"),
	} {
		if err := loadOptionalFile(store, path); err != nil {
			return err
		}
	}

	if err := store.Unmarshal("", out); err != nil {
		return fmt.Errorf("decode %s config: %w", name, err)
	}
	return nil
}

func validateEnv(env string) error {
	switch env {
	case "dev", "demo", "prod":
		return nil
	default:
		return fmt.Errorf("unsupported env %q", env)
	}
}

var grpcServices = map[string]struct{}{
	"ingest": {}, "catalog": {}, "llm": {}, "search": {}, "answer": {}, "epistemic": {},
}

func validateRuntime(service string, runtime Runtime) error {
	if runtime.Log.Level == "" {
		return errors.New("log.level is required")
	}
	if runtime.Log.Format == "" {
		return errors.New("log.format is required")
	}
	if service == "gateway" {
		if runtime.HTTP.Addr == "" {
			return errors.New("http.addr is required for gateway")
		}
		return nil
	}
	if _, ok := grpcServices[service]; ok {
		if runtime.GRPC.Addr == "" {
			return fmt.Errorf("grpc.addr is required for %s", service)
		}
		if runtime.Health.Addr == "" && runtime.HTTP.Addr == "" {
			return errors.New("health.addr or http.addr is required")
		}
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
