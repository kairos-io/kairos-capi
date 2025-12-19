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
	cmd.AddCommand(newInstallCdiCmd())
	cmd.AddCommand(newInstallKubevirtCmd())
	cmd.AddCommand(newInstallCapiCmd())
	cmd.AddCommand(newInstallOsbuilderCmd())
	cmd.AddCommand(newInstallCertManagerCmd())
	cmd.AddCommand(newInstallKairosProviderCmd())

	return cmd
}
