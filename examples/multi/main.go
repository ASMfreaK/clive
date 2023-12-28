package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Southclaws/clive"
	"github.com/urfave/cli/v2"
)

func main() {
	type run struct {
		cli.Command `cli:"usage:this command runs things"` // embedding this is necessary
		FlagHost    string
		FlagPort    int
		FlagDoStuff bool
	}

	type debug struct {
		cli.Command `cli:"name:dbg,usage:this command debugs things"` // embedding this is necessary
		FlagTarget  string
		FlagMethod  string
		FlagTime    time.Duration `cli:"default:1h"`
		FlagForce   bool
	}

	err := clive.Build(
		run{
			Command: cli.Command{
				Action: func(c *cli.Context) error {
					flags, ok := clive.Flags(run{}, c).(run)
					if !ok {
						return errors.New("failed to decode flags")
					}

					fmt.Printf("%+v\n", flags)
					return nil
				},
			},
		},
		debug{
			Command: cli.Command{
				Action: func(c *cli.Context) error {
					flags, ok := clive.Flags(debug{}, c).(debug)
					if !ok {
						return errors.New("failed to decode flags")
					}

					fmt.Println(flags)
					return nil
				},
			},
		}).Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
