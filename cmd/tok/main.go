package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	var tokCmd = &cobra.Command{
		Use:   "tok",
		Short: "Blockchain Go Token",
		Run: func(cmd *cobra.Command, args []string) {
		},
	}

	tokCmd.AddCommand(versionCmd)
	tokCmd.AddCommand(balancesCmd())
	tokCmd.AddCommand(txCmd())

	err := tokCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func incorrectUsageErr() error {
	return fmt.Errorf("incorrect usage")
}
