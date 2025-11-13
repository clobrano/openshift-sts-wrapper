package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
	quiet   bool
)

var rootCmd = &cobra.Command{
	Use:   "openshift-sts-wrapper",
	Short: "OpenShift STS Installation Wrapper",
	Long: `A CLI tool that automates the installation of OpenShift clusters
with AWS Security Token Service (STS) authentication.`,
	Version: "0.1.0",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./openshift-sts-wrapper.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "q", "q", false, "quiet output (errors only)")
}

func getLogLevel() int {
	if quiet {
		return 0 // LevelQuiet
	}
	if verbose {
		return 2 // LevelVerbose
	}
	return 1 // LevelNormal
}

func checkErr(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
