package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	clive "github.com/ASMfreaK/clive2"
	"github.com/urfave/cli/v2"
)

type Start struct {
	*clive.Command `cli:"usage:'stop service'"`
	Service        []string `cli:"positional,usage:'services to use'"`
}

func (start *Start) Action(*cli.Context) error {
	fmt.Printf("start %+v\n", start)
	return nil
}

type Stop struct {
	*clive.Command `cli:"usage:'stop service'"`
	Service        []string `cli:"positional,usage:'services to use'"`
}

func (start *Stop) Action(*cli.Context) error {
	fmt.Printf("stop %+v\n", start)
	return nil
}

type Json struct {
	Value interface{}
}

func (j *Json) MarshalText() ([]byte, error) { return json.Marshal(&j.Value) }

func (j *Json) UnmarshalText(text []byte) error { return json.Unmarshal(text, &j.Value) }

type Role int

const (
	Server Role = iota
	Client
)

var roleStrings = []string{
	Server: "server",
	Client: "client",
}

func (c Role) String() string {
	return roleStrings[c]
}

func (c *Role) UnmarshalText(text []byte) error {
	for i, variant := range roleStrings {
		if string(text) == variant {
			*c = Role(i)
			return nil
		}
	}
	return fmt.Errorf("invalid color: %s", text)
}

func (c Role) MarshalText() ([]byte, error) {
	return []byte(c.String()), nil
}

var AllRoles = []Role{
	Server,
	Client,
}

type SetOption struct {
	*clive.Command `cli:"usage:'configure system'"`

	Name  string `cli:"positional"`
	Value Json   `cli:"positional"`
}

func (one *SetOption) Action(c *cli.Context) error {
	app := one.Root(c).(*App)
	var parent interface{} = nil //one.Parent(c)
	fmt.Printf("set-option %+v app: %+v parent: %+v\n", one, app, parent)
	return nil
}

type App struct {
	*clive.Command `cli:"name:'testCli',usage:'test command'"`

	Subcommands struct {
		*Start
		*Stop
		Config *struct {
			*clive.Command `cli:"name:'config',usage:'configure system'"`

			Subcommands struct {
				*SetOption
			}
		}
	}

	PostgresDsn           string `cli:"default:hello"`
	ProcessSchedule       string `cli:"hidden:true"`
	ApplicationAPIAddress string `cli:"name:api_address"`
}

func main() {
	app := clive.Build(&App{})
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
