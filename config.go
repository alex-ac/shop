package shop

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

const (
	DefaultRegistryName = "default"
)

type ConfigLoadError struct {
	error
	Path string
}

func (e ConfigLoadError) Error() string {
	return fmt.Sprintf("Can't load config (%s): %v", e.error.Error())
}

func NewConfigLoadError(err error, path string) error {
	return ConfigLoadError{
		error: err,
		Path:  path,
	}
}

type ConfigSaveError struct {
	error
	Path string
}

func (e ConfigSaveError) Error() string {
	return fmt.Sprintf("Can't save config (%s): %v", e.error.Error())
}

func NewConfigSaveError(err error, path string) error {
	return ConfigSaveError{
		error: err,
		Path:  path,
	}
}

func wrapConfigError(err *error, path *string, constructor func(error, string) error) {
	if *err != nil {
		*err = constructor(*err, *path)
	}
}

type Config struct {
	DefaultRegistry string `toml:"default_registry,omitempty" comment:"Default registry to use."`
	Cache           string `toml:"cache,omitempty" comment:"Path to the local file cache."`

	Registries map[string]RegistryConfig `toml:"registry,omitempty"`
}

type RegistryConfig struct {
	URL      string                      `toml:"url" comment:"Manifest url."`
	RootRepo RepositoryConfig            `toml:"root_repository" comment:"Main repository settings."`
	Repo     map[string]RepositoryConfig `toml:"repo,omitempty" comment:"Secondar repositories settings."`

	// Local tool configuration
	Admin bool `toml:"admin,omitempty" comment:"Enable admin commands for this registry."`
	Write bool `toml:"write,omitempty" comment:"Enable write commands for this registry."`
}

type RepositoryConfig struct {
	URL  string           `toml:"url" comment:"Repository URL"`
	S3   S3AccessConfig   `toml:"s3,omitempty" comment:"S3 Bucket access config."`
	HTTP HTTPAccessConfig `toml:"http,omitempty" comment:"HTTP access config."`
}

type S3AccessConfig struct {
	// S3 Bucket settings.
	EndpointURL string `toml:"endpoint_url,omitempty" comment:"S3 Endpoint url."`
	Region      string `toml:"region" comment:"AWS region."`
	Bucket      string `toml:"bucket" comment:"S3 Bucket name."`

	// S3 Auth information.
	AWSProfile      string `toml:"aws_profile,omitempty" comment:"AWS profile name."`
	AccessKeyId     string `toml:"access_key_id,omitempty" comment:"AWS Access Key ID."`
	SecretAccessKey string `toml:"secret_access_key,omitempty" comment:"AWS Secret Access Key."`
}

type HTTPAccessConfig struct {
	User       string `toml:"user,omitempty"`
	Password   string `toml:"password,omitempty"`
	ClientCert string `toml:"client_cert,omitempty" comment:"Client Certificate in PEM format."`
}

// Find location of the config file. Should be
// $XDG_CONFIG_HOME/shop/config.toml
func FindConfigFile() (path string, err error) {
	defer wrapConfigError(&err, &path, NewConfigLoadError)

	path, err = os.UserConfigDir()
	if err == nil {
		path = filepath.Join(path, "shop", "config.toml")
	}

	return
}

// Load config from file at path. Returns empty config if file does not exist.
func LoadConfig(path string) (cfg Config, err error) {
	defer wrapConfigError(&err, &path, NewConfigLoadError)

	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		err = nil
		return
	}
	if err != nil {
		return
	}
	defer file.Close()

	decoder := toml.NewDecoder(file)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&cfg)
	return
}

func SaveConfig(cfg Config, path string) (err error) {
	defer wrapConfigError(&err, &path, NewConfigSaveError)

	if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0600)
	if err != nil {
		return
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	err = encoder.Encode(&cfg)

	return
}
