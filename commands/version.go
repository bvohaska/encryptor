package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {

}

var (
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Display this binary's version of Foil",
		Long:  `Foil has versions just like everything else. Yolo.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Version 0.1a")
		},
	}

	authorCmd = &cobra.Command{
		Use:   "authors",
		Short: "Display the authors of Foil",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Authors: Brian Vohaska <bvohaska@gmail.com>")
		},
	}
)