package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Set at build time via:
//
//	go build -ldflags="-X github.com/saitddundar/gordion-vpn/cli/cmd.Version=v0.1.0"
var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print Gordion VPN version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s %s\n",
			styleTitle.Render("Gordion VPN"),
			styleBold.Render(Version),
		)
		fmt.Printf("%s %s/%s\n",
			styleDim.Render("runtime:"),
			runtime.GOOS,
			runtime.GOARCH,
		)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
