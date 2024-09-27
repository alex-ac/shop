package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/alex-ac/shop"
	"github.com/spf13/cobra"
)

var (
	ErrRegistryDoesNotExist = errors.New("Registry does not exist")
)

type PackageCommand struct {
	Arguments *GlobalArguments
}

func NewPackageCommand(args *GlobalArguments) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "package",
		Short: "Manage packages.",
	}

	cmd.AddCommand(
		NewPackageListCommand(args),
	)

	return cmd
}

type PackageListCommand struct {
	Arguments *GlobalArguments

	RegistryName string
}

func NewPackageListCommand(args *GlobalArguments) *cobra.Command {
	c := &PackageListCommand{
		Arguments:    args,
		RegistryName: shop.DefaultRegistryName,
	}

	cmd := &cobra.Command{
		Use:   "ls [prefix]",
		Short: "List packages in registry.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prefix := ""
			if len(args) > 0 {
				prefix = args[0]
			}
			return c.Run(cmd.Context(), prefix)
		},
	}

	cmd.PersistentFlags().StringVarP(&c.RegistryName, "registry", "r", "", "Registry name.")

	return cmd
}

func (c *PackageListCommand) Run(ctx context.Context, prefix string) error {
	cfg, err := c.Arguments.LoadConfig()
	if err != nil {
		return err
	}

	if c.RegistryName == "" {
		c.RegistryName = cfg.DefaultRegistry
	}
	if c.RegistryName == "" {
		c.RegistryName = shop.DefaultRegistryName
	}

	registryConfig, ok := cfg.Registries[c.RegistryName]
	if !ok {
		return fmt.Errorf("%w: %s", ErrRegistryDoesNotExist, c.RegistryName)
	}

	registryClient, err := shop.NewRegistryClient(ctx, registryConfig)
	if err != nil {
		return err
	}

	var output []PackageListOutputItem
	cursor := registryClient.ListPackages(ctx, prefix)
	for {
		pkg, err := cursor.GetNext(ctx)
		if err != nil {
			return err
		}
		if pkg == nil {
			break
		}

		output = append(output, PackageListOutputItemFromPackage(pkg))
	}

	encoder := c.Arguments.OutputFormat.CreateEncoder(os.Stdout)
	return encoder.Encode(output)
}

type PackageListOutputItem struct {
	Name string `json:"name"`
}

func (i PackageListOutputItem) IntoText() ([]byte, error) {
	return []byte(i.Name), nil
}

func PackageListOutputItemFromPackage(pkg *shop.Entry) PackageListOutputItem {
	return PackageListOutputItem{
		Name: pkg.Key,
	}
}
