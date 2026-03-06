package cmd

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Set at build time via:
//
//	go build -ldflags="-X github.com/saitddundar/gordion-vpn/cli/cmd.Version=v0.1.0"
var Version = "dev"

// VersionOutput is the JSON schema for `gordion version --json`
type VersionOutput struct {
	Version string `json:"version"`
	OS      string `json:"os"`
	Arch    string `json:"arch"`
	GoVer   string `json:"go_version"`
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print Gordion VPN version",
	RunE: func(cmd *cobra.Command, args []string) error {
		out := VersionOutput{
			Version: Version,
			OS:      runtime.GOOS,
			Arch:    runtime.GOARCH,
			GoVer:   runtime.Version(),
		}

		if outputJSON {
			data, err := json.MarshalIndent(out, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("%s %s\n",
			styleTitle.Render("Gordion VPN"),
			styleBold.Render(out.Version),
		)
		fmt.Printf("%s %s/%s (%s)\n",
			styleDim.Render("runtime:"),
			out.OS,
			out.Arch,
			out.GoVer,
		)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
