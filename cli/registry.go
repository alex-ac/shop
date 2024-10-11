package cli

import (
	"context"
	"errors"
	"os"

	"github.com/alex-ac/shop"
	"github.com/spf13/cobra"
)

func NewRegistryCommand(args *GlobalArguments) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "registry",
		Short: "Manage registries.",
	}

	cmd.AddCommand(
		NewRegistryInitCommand(args),
		NewRegistryAddCommand(args),
		NewRegistryListCommand(args),
		NewRegistryDeleteCommand(args),
	)

	return cmd
}

type RegistryAddCommand struct {
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

func NewRegistryAddCommand(args *GlobalArguments) *cobra.Command {
	c := &RegistryAddCommand{
		Arguments:    args,
		RegistryName: shop.DefaultRegistryName,
	}

	cmd := &cobra.Command{
		Use:   "add [-n name] [-a] [-w] [options] url",
		Short: "Add registry to configuration.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.Run(cmd.Context(), args[0])
		},
	}

	cmd.PersistentFlags().StringVarP(&c.RegistryName, "name", "n", "", "Registry name.")
	cmd.PersistentFlags().BoolVarP(&c.Admin, "admin", "a", false, "Enable registry administration commands (requires read-write access to the bucket).")
	cmd.PersistentFlags().BoolVarP(&c.Admin, "write", "w", false, "Enable registry write commands (requires read-write access to the bucket).")

	return cmd
}

func (c *RegistryAddCommand) Run(ctx context.Context, url string) error {
	registryConfig := shop.RegistryConfig{
		RootRepo: shop.RepositoryConfig{
			URL: url,
		},
	}

	registryClient, err := shop.NewRegistry(ctx, registryConfig)
	if err != nil {
		return err
	}

	registryManifest, err := registryClient.GetManifest(ctx)
	if err != nil {
		return err
	}

	registryConfig = registryClient.GetConfig()

	if c.RegistryName == "" {
		c.RegistryName = registryManifest.Name
	}

	cfg, err := c.Arguments.LoadConfig()
	if err != nil {
		return err
	}

	err = cfg.AddRegistry(c.RegistryName, registryConfig)
	if err != nil {
		return err
	}

	cfg.Registries[c.RegistryName] = registryConfig

	return c.Arguments.SaveConfig(cfg)
}

type RegistryListCommand struct {
	Arguments *GlobalArguments
}

type RegistryListOutputItem struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	Admin     bool   `json:"admin"`
	Write     bool   `json:"write"`
	IsDefault bool   `json:"default"`
}

func (i RegistryListOutputItem) IntoText() (d []byte, err error) {
	if i.IsDefault {
		d = []byte("* ")
	} else {
		d = []byte("  ")
	}
	d = append(d, i.Name...)
	d = append(d, byte(' '))
	d = append(d, i.URL...)

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
		Name:      name,
		URL:       registry.RootRepo.URL,
		Admin:     registry.Admin,
		Write:     registry.Write,
		IsDefault: isDefault,
	}
}

func NewRegistryListCommand(args *GlobalArguments) *cobra.Command {
	c := &RegistryListCommand{
		Arguments: args,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List registry configurations.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.Run(cmd.Context())
		},
	}

	return cmd
}

func (c *RegistryListCommand) Run(ctx context.Context) error {
	cfg, err := c.Arguments.LoadConfig()
	if err != nil {
		return err
	}

	if cfg.DefaultRegistry == "" {
		cfg.DefaultRegistry = shop.DefaultRegistryName
	}

	encoder := c.Arguments.OutputFormat.CreateEncoder(os.Stdout)
	output := make([]RegistryListOutputItem, 0, len(cfg.Registries))
	for name, registry := range cfg.Registries {
		output = append(output, RegistryListOutputItemFromRegistry(registry, name, name == cfg.DefaultRegistry))
	}

	return encoder.Encode(output)
}

type RegistryDeleteCommand struct {
	Arguments *GlobalArguments
}

func NewRegistryDeleteCommand(args *GlobalArguments) *cobra.Command {
	c := &RegistryDeleteCommand{
		Arguments: args,
	}

	cmd := &cobra.Command{
		Use:   "delete name",
		Short: "Delete registry configuration.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.Run(cmd.Context(), args[0])
		},
	}

	return cmd
}

func (c *RegistryDeleteCommand) Run(ctx context.Context, name string) error {
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

func CompleteRegistryFlag(cmd *cobra.Command, argv []string, toComplete string) (variants []string, directive cobra.ShellCompDirective) {
	_ = cmd.ParseFlags(argv)
	args := DefaultGlobalArguments
	args.Config = cmd.Flag("config").Value.String()
	cfg, err := args.LoadConfig()
	if err != nil {
		directive = cobra.ShellCompDirectiveError
		return
	}

	for name, _ := range cfg.Registries {
		variants = append(variants, name)
	}

	return
}

type RegistryInitCommand struct {
	Arguments    *GlobalArguments
	Name         string
	ManifestName string
}

func NewRegistryInitCommand(args *GlobalArguments) *cobra.Command {
	c := &RegistryInitCommand{
		Arguments: args,
	}

	cmd := &cobra.Command{
		Use:   "init -N manifest-name [-n name] url",
		Short: "Initialize new registry in given repository.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.Run(cmd.Context(), args[0])
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if !cmd.Flag("name").Changed {
				c.Name = c.ManifestName
			}
		},
	}

	cmd.PersistentFlags().StringVarP(&c.ManifestName, "manifest-name", "N", "", "Name for the repository in manifest.")
	cmd.PersistentFlags().StringVarP(&c.Name, "name", "n", "", "Name for the repository in config.")
	cmd.MarkPersistentFlagRequired("manifest-name")

	return cmd
}

func (c *RegistryInitCommand) Run(ctx context.Context, url string) error {
	cfg, err := c.Arguments.LoadConfig()
	if err != nil {
		return err
	}

	repoConfig := shop.RepositoryConfig{
		URL:   url,
		Admin: true,
		Write: true,
	}

	repo, err := shop.NewRepository(ctx, repoConfig)
	if err != nil {
		return err
	}

	repoManifest, err := repo.GetManifest(ctx)
	if err != nil {
		return err
	}

	registryConfig := shop.RegistryConfig{
		URL:      repoManifest.URL,
		RootRepo: repoConfig,
		Admin:    true,
		Write:    true,
	}

	err = cfg.AddRegistry(c.Name, registryConfig)
	if err != nil && !errors.Is(err, shop.ErrRegistryConfigExists) {
		return err
	}

	registry, err := shop.NewRegistry(ctx, registryConfig)
	if err != nil {
		return err
	}

	err = registry.Initialize(ctx, c.ManifestName)
	if err != nil {
		return err
	}

	return c.Arguments.SaveConfig(cfg)
}
