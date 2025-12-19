package main

import (
	"github.com/spf13/cobra"
)

func newInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install components",
		Long:  "Install various components on the cluster",
	}

	cmd.AddCommand(newInstallCalicoCmd())
	cmd.AddCommand(newInstallKubevirtCmd())
	cmd.AddCommand(newInstallCapiCmd())
	cmd.AddCommand(newInstallOsbuilderCmd())

	return cmd
}
