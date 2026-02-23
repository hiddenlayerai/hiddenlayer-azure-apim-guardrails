package main

import (
	"os"

	"github.com/hiddenlayerai/hiddenlayer-azure-apim-guardrails/cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
