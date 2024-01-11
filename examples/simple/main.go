package main

import (
	"fmt"
	"log"
	"os"

	clive "github.com/ASMfreaK/clive2"
	"github.com/urfave/cli/v2"
)

func main() {
	type app struct {
		*clive.Command `cli:"usage:'this command does a, b and c'"` // embedding this is necessary
		Run            clive.RunFunc
		FlagHost       string
		FlagPort       int
		FlagDoStuff    bool
	}
	err := clive.Build(&app{Run: func(c *clive.Command, ctx *cli.Context) error {
		a := c.Current(ctx).(*app)
		fmt.Println(a.FlagHost)
		fmt.Println(a.FlagPort)
		fmt.Println(a.FlagDoStuff)
		return nil
	}}).Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
