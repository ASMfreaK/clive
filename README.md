# **CLI**, made **V**ery **E**asy!, second overhaul (clive2)

This compacts the huge declaration of a `cli.App` into a declarative, (mostly) compile-time checked and automated
generator based on struct tags.

## Motivation

Aside from `urfave/cli` declarations getting rather huge, there tend to be a _ton_ of duplication which not only is it
annoying to write and maintain, it can also introduce subtle bugs due to the usage of string literals instead of typed
objects that are checked by tools and the compiler.

It's also fun to mess around with structs, reflection and tags!

## Usage

A single struct can declare your entire application, then at run-time all you have to do is bind the `Action`, `Before`,
`After`, etc. fields to your functions.

### One Command

A single-command instance will make all the flags global and assign the action function to the root `App` object:

```go
type app struct {
	*clive.Command `cli:"usage:'this command does a, b and c'"` // embedding this is necessary
	Run            clive.RunFunc
	FlagHost       string
	FlagPort       int
	FlagDoStuff    bool
}
func (*app) Action(ctx *cli.Context){
	a := c.Current(ctx).(*app)
	fmt.Println(a.FlagHost)
	fmt.Println(a.FlagPort)
	fmt.Println(a.FlagDoStuff)
	return nil
}
func main() {
	err := clive.Build(&app{}).Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
```

This will produce a `cli.App` object that looks something like this:

```go
&cli.App{
    Name:                 "application-thing",
    Usage:                "this command does a, b and c",
    Version:              "60f4851-master (2018-10-25T16:33:27+0000)",
    Description:          "the contents of the APP_DESCRIPTION environment variable",
    Flags:       {
        cli.StringFlag{
            Name:        "flag-host",
            EnvVar:      "FLAG_HOST",
        },
        cli.StringFlag{
            Name:        "flag-port",
            EnvVar:      "FLAG_PORT",
        },
        cli.StringFlag{
            Name:        "do-stuff",
            EnvVar:      "DO_STUFF",
        },
    },
}
```

The flag names and environment variables have been filled in automatically and converted to their respective cases
(kebab and screaming-snake).

In the `Action` function, the `flags := clive.Flags(c, run{}).(run)` line is responsible for taking the `*cli.Context`
parameter that is passed in by `cli`, extracting the flag values and returning a value that you can safely cast to the
original struct type.

This allows you to access the flag values in a _type safe_ way without relying on string-literals that can be mistyped.

### Multiple Command

If you supply multiple structs to `clive.Build`, they will form an `App` with `Command`s instead of an `App` with an
`Action` function and global `Flags`.

```go
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

					fmt.Println(flags)
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
```

## Available Tags

The `cli` struct tag group can be used to tweak flags. In the example above, it's used on the `cli.Command` field to set
the "usage" text but it can also be used on `Flag` fields.

The format is:

```
`cli:"key:value,key:'value',key:value"`
```

Single quotes (`''`) are allowed to escape the comma (`,`) character.

The available tag names are:

- `name`: override the flag name
- `usage`: set the usage text for the flag
- `hidden`: hide the flag
- `default`: set the default value
- `required`: set the required flag

- `positional`: converts flag into a positional argument (taken from `ctx.Args()`)

The only tag used for the top-level `App` is `usage` which must be applied to the embedded `cli.Command` struct.



