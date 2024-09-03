package main

import (
	"fadingrose/rosy-nigh/fuzz"
	"fadingrose/rosy-nigh/log"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	var contractFolder string
	var verbose bool
	var debugSession bool
	var debug bool
	fuzzCli := &cli.App{
		Name:  "rosy-nigh",
		Usage: "A fuzzing tool for Ethereum Smart Contract bytecode",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "verbose",
				Aliases:     []string{"v"},
				Usage:       "Enable verbose output",
				Value:       false,
				Destination: &verbose,
			},
			&cli.BoolFlag{
				Name:        "debug",
				Aliases:     []string{"d"},
				Usage:       "Enable debug log",
				Value:       false,
				Destination: &debug,
			},
			&cli.BoolFlag{
				Name:        "session",
				Aliases:     []string{"s"},
				Usage:       "Enable debug session",
				Value:       false,
				Destination: &debugSession,
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "fuzz",
				Usage: "Fuzz a smart contract",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "contract folder",
						Aliases:     []string{"i"},
						Usage:       "Folder containing the smart contract bytecode to fuzz",
						Required:    true,
						Value:       "",
						Destination: &contractFolder,
					},
				},
				Action: func(c *cli.Context) error {
					if debug {
						enableDebugLogging()
					} else {
						enableVerboseLogging()
					}
					err := fuzz.Execute(contractFolder, debugSession)
					if err != nil {
						fmt.Println("runtime err", err)
					}
					return nil
				},
			},
		},
	}
	fuzzCli.Run(os.Args)
}

// enableVerboseLogging enables verbose output to terminal
func enableVerboseLogging() {
	log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stderr, log.LevelTrace, true)))
}

func enableDebugLogging() {
	log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stderr, log.LevelDebug, true)))
}
