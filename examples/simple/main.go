package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/Southclaws/clive"
	"github.com/urfave/cli/v2"
)

func main() {
	type app struct {
		cli.Command `cli:"usage:'this command does a, b and c'"` // embedding this is necessary
		FlagHost    string
		FlagPort    int
		FlagDoStuff bool
	}
	err := clive.Build(app{
		Command: cli.Command{
			Action: func(c *cli.Context) error {
				flags, ok := clive.Flags(app{}, c).(app)
				if !ok {
					return errors.New("failed to decode flags")
				}

				fmt.Println("Flags:")
				fmt.Println(flags.FlagHost)
				fmt.Println(flags.FlagPort)
				fmt.Println(flags.FlagDoStuff)

				return nil
			},
		},
	}).Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
