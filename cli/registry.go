package cli

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/alex-ac/shop"
)

type RegistryCommand struct {
	Subcommands
	Arguments *GlobalArguments
}

func NewRegistryCommand(prefix string, args *GlobalArguments) *RegistryCommand {
	r := &RegistryCommand{
		Subcommands: NewSubcommands(prefix, "registry", "Manage registries."),
		Arguments:   args,
	}

	r.Register(NewRegistryAddCommand(r.SubPrefix(), args), "")
	r.Register(NewRegistryListCommand(r.SubPrefix(), args), "")
	r.Register(NewRegistryDeleteCommand(r.SubPrefix(), args), "")

	return r
}

type RegistryAddCommand struct {
	BaseCommand
	Arguments *GlobalArguments

	RegistryName string
	Admin        bool
	Write        bool

	EndpointUrl     string
	Region          string
	AWSProfile      string
	AccessKeyId     string
	SecretAccessKey string
}

func NewRegistryAddCommand(prefix string, args *GlobalArguments) *RegistryAddCommand {
	c := &RegistryAddCommand{
		BaseCommand:  NewBaseCommand(prefix, "add", "Add new registry configuration."),
		Arguments:    args,
		RegistryName: shop.DefaultRegistryName,
	}

	c.FlagSet.StringVar(&c.RegistryName, "name", shop.DefaultRegistryName, "Registry name.")
	c.FlagSet.BoolVar(&c.Admin, "admin", false, "Enable registry administration commands (requires read-write access to the bucket).")
	c.FlagSet.BoolVar(&c.Admin, "write", false, "Enable registry write commands (requires read-write access to the bucket).")

	c.FlagSet.StringVar(&c.EndpointUrl, "aws-endpoint-url", "", "URL of S3 endpoint (if not AWS).")
	c.FlagSet.StringVar(&c.Region, "aws-region", "", "AWS region.")
	c.FlagSet.StringVar(&c.AWSProfile, "aws-profile", "", "Name of the AWS cli profile.")
	c.FlagSet.StringVar(&c.AccessKeyId, "aws-access-key-id", "", "AWS access key id.")
	c.FlagSet.StringVar(&c.SecretAccessKey, "aws-secret-access-key", "", "AWS secret access key.")

	return c
}

func (c *RegistryAddCommand) Run(ctx context.Context, fs *flag.FlagSet, _ ...any) error {
	if fs.NArg() != 1 {
		return fmt.Errorf("%w: %s %s requires exactly 1 argument (S3 Bucket name).\n", ErrUsage, c.Prefix, c.Name())
	}
	bucket := fs.Arg(0)

	cfg, err := c.Arguments.LoadConfig()
	if err != nil {
		return err
	}

	if cfg.Registries == nil {
		cfg.Registries = map[string]shop.RegistryConfig{}
	}

	if _, registryExists := cfg.Registries[c.RegistryName]; registryExists {
		return fmt.Errorf("Can't create registry '%s' because it is already exists in config.", c.RegistryName)
	}

	cfg.Registries[c.RegistryName] = shop.RegistryConfig{
		EndpointURL:     c.EndpointUrl,
		Region:          c.Region,
		Bucket:          bucket,
		AWSProfile:      c.AWSProfile,
		AccessKeyId:     c.AccessKeyId,
		SecretAccessKey: c.SecretAccessKey,
		Admin:           c.Admin,
		Write:           c.Write,
	}

	return c.Arguments.SaveConfig(cfg)
}

type RegistryListCommand struct {
	BaseCommand
	Arguments *GlobalArguments
}

type RegistryListOutputItem struct {
	Name        string `json:"name"`
	Bucket      string `json:"bucket"`
	Region      string `json:"region,omitempty"`
	EndpointURL string `json:"endpoint_url,omitempty"`
	AWSProfile  string `json:"aws_profile,omitempty"`
	Admin       bool   `json:"admin"`
	Write       bool   `json:"write"`
	IsDefault   bool   `json:"default"`
}

func (i RegistryListOutputItem) MarshalText() (d []byte, err error) {
	if i.IsDefault {
		d = []byte("* ")
	} else {
		d = []byte("  ")
	}
	d = append(d, i.Name...)
	d = append(d, " bucket="...)
	d = append(d, i.Bucket...)
	if i.Region != "" {
		d = append(d, " region="...)
		d = append(d, i.Region...)
	}

	if i.EndpointURL != "" {
		d = append(d, " endpoint_url="...)
		d = append(d, i.EndpointURL...)
	}

	if i.AWSProfile != "" {
		d = append(d, " aws_profile="...)
		d = append(d, i.AWSProfile...)
	}

	if i.Admin {
		d = append(d, " +admin"...)
	}

	if i.Write {
		d = append(d, " +write"...)
	}
	return
}

func RegistryListOutputItemFromRegistry(registry shop.RegistryConfig, name string, isDefault bool) RegistryListOutputItem {
	return RegistryListOutputItem{
		Name:        name,
		Bucket:      registry.Bucket,
		Region:      registry.Region,
		EndpointURL: registry.EndpointURL,
		AWSProfile:  registry.AWSProfile,
		Admin:       registry.Admin,
		Write:       registry.Write,
		IsDefault:   isDefault,
	}
}

func NewRegistryListCommand(prefix string, args *GlobalArguments) *RegistryListCommand {
	c := &RegistryListCommand{
		BaseCommand: NewBaseCommand(prefix, "list", "List registry configurations."),
		Arguments:   args,
	}

	return c
}

func (c *RegistryListCommand) Run(ctx context.Context, fs *flag.FlagSet, _ ...any) error {
	cfg, err := c.Arguments.LoadConfig()
	if err != nil {
		return err
	}

	if cfg.DefaultRegistry == "" {
		cfg.DefaultRegistry = shop.DefaultRegistryName
	}

	if encoder := c.Arguments.OutputFormat.CreateEncoder(os.Stdout); encoder != nil {
		output := make([]RegistryListOutputItem, 0, len(cfg.Registries))
		for name, registry := range cfg.Registries {
			output = append(output, RegistryListOutputItemFromRegistry(registry, name, name == cfg.DefaultRegistry))
		}

		return encoder.Encode(output)
	}

	return nil
}

type RegistryDeleteCommand struct {
	BaseCommand
	Arguments *GlobalArguments
}

func NewRegistryDeleteCommand(prefix string, args *GlobalArguments) *RegistryDeleteCommand {
	c := &RegistryDeleteCommand{
		BaseCommand: NewBaseCommand(prefix, "delete", "Delete registry configuration."),
		Arguments:   args,
	}

	return c
}

func (c *RegistryDeleteCommand) Run(ctx context.Context, fs *flag.FlagSet, _ ...any) error {
	if fs.NArg() != 1 {
		return fmt.Errorf("%w: %s %s requires exactly 1 argument (registry name).\n", ErrUsage, c.Prefix, c.Name())
	}
	name := fs.Arg(0)

	cfg, err := c.Arguments.LoadConfig()
	if err != nil {
		return err
	}

	if cfg.Registries == nil {
		return nil
	}

	delete(cfg.Registries, name)

	return c.Arguments.SaveConfig(cfg)
}
