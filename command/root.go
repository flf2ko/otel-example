package command

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "otel-example",
	Short: "otel-example",
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
	},
}

func Execute() error {
	rootCmd.AddCommand(NewServerCmd())
	return rootCmd.Execute()
}
