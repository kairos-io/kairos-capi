package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	defaultClusterName = "kairos-capi-test"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "kubevirt-env",
		Short: "KubeVirt Local Testing Environment CLI",
		Long:  "A CLI tool for managing local KubeVirt testing environments",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initializeConfig()
		},
	}

	rootCmd.PersistentFlags().String("cluster-name", defaultClusterName, "Cluster name (can also be set via CLUSTER_NAME env var)")
	viper.BindPFlag("cluster-name", rootCmd.PersistentFlags().Lookup("cluster-name"))
	viper.BindEnv("cluster-name", "CLUSTER_NAME")

	rootCmd.AddCommand(newCreateTestClusterCmd())
	rootCmd.AddCommand(newInstallCalicoCmd())
	rootCmd.AddCommand(newInstallKubevirtCmd())
	rootCmd.AddCommand(newInstallCapiCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func initializeConfig() error {
	viper.SetEnvPrefix("")
	viper.AutomaticEnv()
	return nil
}

func getClusterName() string {
	return viper.GetString("cluster-name")
}

func getWorkDir() string {
	clusterName := getClusterName()
	return filepath.Join(".work-kubevirt-" + clusterName)
}

func getKubeconfigPath() string {
	return filepath.Join(getWorkDir(), "kubeconfig")
}

func getKubectlContext() string {
	clusterName := getClusterName()
	return fmt.Sprintf("kind-%s", clusterName)
}
