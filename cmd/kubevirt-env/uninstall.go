package main

import (
	"github.com/spf13/cobra"
)

func newUninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall components",
		Long:  "Uninstall various components from the cluster",
	}

	cmd.AddCommand(newUninstallCalicoCmd())
	cmd.AddCommand(newUninstallLocalPathCmd())
	cmd.AddCommand(newUninstallCdiCmd())
	cmd.AddCommand(newUninstallKubevirtCmd())
	cmd.AddCommand(newUninstallCapiCmd())
	cmd.AddCommand(newUninstallCapkCmd())
	cmd.AddCommand(newUninstallOsbuilderCmd())
	cmd.AddCommand(newUninstallCertManagerCmd())
	cmd.AddCommand(newUninstallKairosProviderCmd())

	return cmd
}
