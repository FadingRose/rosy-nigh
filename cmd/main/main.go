package main

import (
	"bytes"
	"fadingrose/rosy-nigh/core/vm"
	"fadingrose/rosy-nigh/fuzz"
	"fadingrose/rosy-nigh/log"
	"fadingrose/rosy-nigh/terminal"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	var contractFolder string
	var verbose bool
	var debugSession bool
	var debug bool
	var onchainAddress string

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
				Name:  "local",
				Usage: "Fuzz a smart contract locally",
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
			{
				Name:  "onchain",
				Usage: "Fuzz a smart contract onchain",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "address",
						Aliases:     []string{"a"},
						Usage:       "Address of the smart contract to fuzz",
						Required:    true,
						Value:       "",
						Destination: &onchainAddress,
					},
				},
				Action: func(c *cli.Context) error {
					if debug {
						enableDebugLogging()
					} else {
						enableVerboseLogging()
					}
					cacheFolder, err := fuzz.PrepareOnchainCache(onchainAddress)
					if err != nil {
						return fmt.Errorf("failed to prepare onchain cache: %w", err)
					}
					err = fuzz.Execute(cacheFolder, debugSession)
					if err != nil {
						fmt.Println("runtime err", err)
					}
					return nil
				},
			},
			{
				Name:  "debug",
				Usage: "Debug a start Fuzzing server",
				Action: func(c *cli.Context) error {
					client := MockClient{}
					stdin := os.Stdin
					term := terminal.NewTerminal(&client, stdin)
					return term.Run()
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

type MockClient struct{}

func (m *MockClient) RegExpand(pc uint64) (string, error) {
	return "mock reg expand", nil
}

func (m *MockClient) RegOpcode(op vm.OpCode) (string, error) {
	return "mock opcode", nil
}

type StringReader struct {
	buf *bytes.Buffer
}

func newStringReader() *StringReader {
	return &StringReader{buf: new(bytes.Buffer)}
}

func (sr *StringReader) WriteString(s string) (int, error) {
	return sr.buf.WriteString(s)
}

func (sr *StringReader) Read(p []byte) (int, error) {
	return sr.buf.Read(p)
}
