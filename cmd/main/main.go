package main

import (
	"fadingrose/rosy-nigh/fuzz"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	var contractFolder string

	fuzzCli := &cli.App{
		Name:  "rosy-nigh",
		Usage: "A fuzzing tool for Ethereum Smart Contract bytecode",
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
					err := fuzz.Execute(contractFolder)
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
