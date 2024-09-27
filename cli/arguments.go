package cli

import (
	"github.com/alex-ac/shop"
	"github.com/spf13/cobra"
)

func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "shop",
		Short: "A package registry/deployment system for development tools.",
		Long: `A CIPD inspired package registry/deployer.

Shop is a package registry/deployment system for development tools.`,
	}

	return rootCmd
}

type GlobalArguments struct {
	Config       string
	OutputFormat OutputFormat
}

var DefaultGlobalArguments = GlobalArguments{
	OutputFormat: DefaultOutputFormat,
}

func (a *GlobalArguments) Setup(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&a.Config, "config", "f", a.Config, "Path to the config file to use.")
	cmd.MarkPersistentFlagFilename("config", "toml")
	cmd.PersistentFlags().VarP(TextVar{&a.OutputFormat}, "output-format", "o", "Output format.")
	cmd.RegisterFlagCompletionFunc("output-format", func(cmd *cobra.Command, args []string, toComplete string) (variants []string, directive cobra.ShellCompDirective) {
		for format, _ := range AllOutputFormats {
			variants = append(variants, string(format))
		}
		return
	})
}

func (a *GlobalArguments) ResolveConfig() (err error) {
	if a.Config == "" {
		a.Config, err = shop.FindConfigFile()
	}

	return
}

func (a *GlobalArguments) LoadConfig() (cfg shop.Config, err error) {
	err = a.ResolveConfig()

	if err == nil {
		cfg, err = shop.LoadConfig(a.Config)
	}

	return
}

func (a *GlobalArguments) SaveConfig(cfg shop.Config) (err error) {
	err = a.ResolveConfig()

	if err == nil {
		err = shop.SaveConfig(cfg, a.Config)
	}

	return
}
