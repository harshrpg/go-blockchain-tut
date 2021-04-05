package main

import (
	"fmt"
	"os"

	"github.com/harshrpg/go-blockchain-tut/fs"
	"github.com/spf13/cobra"
)

const flagDataDir = "datadir"
const flagPort = "port"

func main() {
	var tokCmd = &cobra.Command{
		Use:   "tok",
		Short: "Blockchain Go Token",
		Run: func(cmd *cobra.Command, args []string) {
		},
	}

	tokCmd.AddCommand(versionCmd)
	tokCmd.AddCommand(balancesCmd())
	tokCmd.AddCommand(runCmd())
	tokCmd.AddCommand(migrateCmd())

	err := tokCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func incorrectUsageErr() error {
	return fmt.Errorf("incorrect usage")
}

func addDefaultRequiredFlags(cmd *cobra.Command) {
	cmd.Flags().String(flagDataDir, "", "Absolute path to the node data dir where the DB will/is stored")
	cmd.MarkFlagRequired(flagDataDir)
}

func getDataDirFromCmd(cmd *cobra.Command) string {
	dataDir, _ := cmd.Flags().GetString(flagDataDir)

	return fs.ExpandPath(dataDir)
}
