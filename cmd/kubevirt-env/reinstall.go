package main

import (
	"github.com/spf13/cobra"
)

func newReinstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reinstall",
		Short: "Reinstall components",
		Long:  "Uninstall and then reinstall components on the cluster",
	}

	cmd.AddCommand(newReinstallCalicoCmd())
	cmd.AddCommand(newReinstallKubevirtCmd())
	cmd.AddCommand(newReinstallCapiCmd())

	return cmd
}
