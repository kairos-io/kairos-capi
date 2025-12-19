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
	cmd.AddCommand(newUninstallKubevirtCmd())
	cmd.AddCommand(newUninstallCapiCmd())
	cmd.AddCommand(newUninstallOsbuilderCmd())

	return cmd
}
