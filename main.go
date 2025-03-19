package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "zed-prompts",
		Usage: "Import and export prompts from LMDB database",
		Commands: []*cli.Command{
			{
				Name:  "export",
				Usage: "Export prompts from LMDB to JSON",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "db",
						Aliases:  []string{"d"},
						Value:    dbPath(),
						Usage:    "LMDB database path",
						Required: false,
					},
					&cli.StringFlag{
						Name:     "output",
						Aliases:  []string{"o"},
						Usage:    "Output JSON file (use '-' for stdout)",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					return runExport(c.String("db"), c.String("output"))
				},
			},
			{
				Name:  "import",
				Usage: "Import prompts from JSON to LMDB",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "input",
						Aliases:  []string{"i"},
						Usage:    "Input JSON file (use '-' for stdin)",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "db",
						Aliases:  []string{"d"},
						Value:    dbPath(),
						Usage:    "LMDB database path",
						Required: false,
					},
				},
				Action: func(c *cli.Context) error {
					return runImport(c.String("input"), c.String("db"))
				},
			},
			{
				Name:  "list",
				Usage: "List prompts",
				Flags: []cli.Flag{},
				Action: func(c *cli.Context) error {
					return listMetadata()
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Printf("error: %+v\n", err)
	}
}

func runExport(dbPath string, output string) error {
	return export(
		dbPath,
		output,
	)
}

func runImport(input string, dbPath string) error {
	os.MkdirAll(dbPath, 0755)
	return importJSON(
		input,
		dbPath,
	)
}
