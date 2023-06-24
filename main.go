package main

import (
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "easy-static",
		Usage: "build static websites from a bunch of unstructured docs",
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:     "input",
				Usage:    "input directory",
				Required: true,
			},
			&cli.PathFlag{
				Name:     "config",
				Usage:    "config directory",
				Required: false,
			},
			&cli.PathFlag{
				Name:     "output",
				Usage:    "output directory",
				Required: true,
			},
			&cli.BoolFlag{
				Name:   "watch",
				Value:  false,
				Hidden: true,
			},
		},
		Action: func(cCtx *cli.Context) error {
			fmt.Println("boom! I say!")
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
