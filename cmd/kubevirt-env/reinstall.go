package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func newReinstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reinstall",
		Short: "Reinstall components",
		Long:  "Uninstall and then reinstall components on the cluster",
	}

	cmd.AddCommand(newReinstallCalicoCmd())
	cmd.AddCommand(newReinstallLocalPathCmd())
	cmd.AddCommand(newReinstallCdiCmd())
	cmd.AddCommand(newReinstallKubevirtCmd())
	cmd.AddCommand(newReinstallCapiCmd())
	cmd.AddCommand(newReinstallCapkCmd())
	cmd.AddCommand(newReinstallOsbuilderCmd())
	cmd.AddCommand(newReinstallCertManagerCmd())
	cmd.AddCommand(newReinstallKairosProviderCmd())

	return cmd
}

func newReinstallCertManagerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cert-manager",
		Short: "Reinstall cert-manager",
		Long:  "Uninstall and reinstall cert-manager",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := uninstallCertManager(); err != nil {
				return fmt.Errorf("failed to uninstall cert-manager: %w", err)
			}
			// Small delay to ensure namespace is fully deleted
			fmt.Println("Waiting for namespace cleanup...")
			time.Sleep(3 * time.Second)
			return installCertManager()
		},
	}
	return cmd
}
