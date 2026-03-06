package cmd

import (
	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart Gordion VPN (down + up)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := downCmd.RunE(cmd, args); err != nil {
			printWarn("Could not stop: " + err.Error())
		}
		return upCmd.RunE(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(restartCmd)
}
