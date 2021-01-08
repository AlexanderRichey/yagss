package main

import (
	"log"

	"github.com/spf13/cobra"

	"github.com/AlexanderRichey/yasst/internal/builder"
	"github.com/AlexanderRichey/yasst/internal/proj"
	"github.com/AlexanderRichey/yasst/internal/server"
)

func main() {
	log.SetFlags(0)

	cmdNew := &cobra.Command{
		Use:   "new <directory name>",
		Short: "Create a new yagss site",
		Long:  `create a new yagss site in the current working directory with the given name`,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			err := proj.New(args[0])
			if err != nil {
				log.Fatal(err)
			}
		},
	}

	cmdBuild := &cobra.Command{
		Use:   "build",
		Short: "Build the current yagss site",
		Long: `build the current yagss site using the config.toml file
in the current working directory.`,
		Run: func(cmd *cobra.Command, args []string) {
			c, err := builder.ReadConfig()
			if err != nil {
				log.Fatal(err)
			}

			b, err := builder.New(c, nil)
			if err != nil {
				log.Fatal(err)
			}

			err = b.Build()
			if err != nil {
				log.Fatal(err)
			}
		},
	}

	var port int
	cmdServe := &cobra.Command{
		Use:   "serve",
		Short: "Serve the current yagss site and auto build when files change",
		Long: `serve the build directory of the current yagss site and
rebuild when source files change`,
		Run: func(cmd *cobra.Command, args []string) {
			c, err := builder.ReadConfig()
			if err != nil {
				log.Fatal(err)
			}

			err = server.Start(c, port)
			if err != nil {
				log.Fatal(err)
			}
		},
	}
	cmdServe.Flags().IntVar(&port, "port", 3000, "default port")

	rootCmd := &cobra.Command{Use: "yagss"}
	rootCmd.AddCommand(cmdNew, cmdBuild, cmdServe)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
