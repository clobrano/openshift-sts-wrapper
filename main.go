package main

import (
	"os"

	"github.com/clobrano/openshift-sts-wrapper/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
