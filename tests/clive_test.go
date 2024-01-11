package clive_test

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	clive "github.com/ASMfreaK/clive2"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
)

type ColorT int

const (
	Red ColorT = iota
	Green
	Blue
)

var colorStrings = []string{
	Red:   "Red",
	Green: "Green",
	Blue:  "Blue",
}

func (c ColorT) String() string {
	return colorStrings[c]
}

func (c *ColorT) UnmarshalText(text []byte) error {
	for i, variant := range colorStrings {
		if string(text) == variant {
			*c = ColorT(i)
			return nil
		}
	}
	return fmt.Errorf("invalid color: %s", text)
}

func (c ColorT) MarshalText() ([]byte, error) {
	return []byte(c.String()), nil
}

var AllColorT = []ColorT{
	Red,
	Green,
	Blue,
}

type Start struct {
	*clive.Command `cli:"usage:'start service'"`
	Service        []string `cli:"positional,usage:'services to use',required:false"`
}

func (start *Start) Action(*cli.Context) error {
	fmt.Printf("start %+v\n", start)
	return nil
}

type Stop struct {
	*clive.Command `cli:"name:'stops',usage:'stop service'"`
	Service        []string `cli:"positional,usage:'services to use',required:false"`
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
	*clive.Command `cli:"usage:'set config option NAME from json VALUE'"`

	Name  string `cli:"positional"`
	Value Json   `cli:"positional"`
}

func (one *SetOption) Action(c *cli.Context) error {
	fmt.Printf("stop %+v\n", one)
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

			ExpensiveFieldToCompute *int `cli:"-"`
		}
	}

	PostgresDsn           string `cli:"default:hello"`
	ProcessSchedule       string `cli:"hidden:true"`
	ApplicationAPIAddress string `cli:"name:api_address"`
	Uints64               []uint64
	Color                 string `cli:"default:Blue"`
}

func (*App) Version() string {
	return "some-version-string"
}

// func TestQueue(t *testing.T) {
// 	rapid.Check(t, func(t *rapid.T) {

// 		rapid.Generator(

// 		)
// 	})
// }

func TestBuild(t *testing.T) {
	type test struct {
		objs  []interface{}
		wantC *cli.App
	}
	tests := []test{}

	// stub := func(*cli.Context) error { return nil }

	app := &App{}
	tests = append(tests, test{
		[]interface{}{
			app,
		},
		&cli.App{
			Name: "tests.test",
			// HelpName: "clive.test",
			Usage:   "test command",
			Version: "some-version-string",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "postgres-dsn",
					EnvVars: []string{"POSTGRES_DSN"},
					Value:   "hello",
				},
				&cli.StringFlag{
					Name:    "process-schedule",
					EnvVars: []string{"PROCESS_SCHEDULE"},
					Hidden:  true,
				},
				&cli.StringFlag{
					Name:    "api-address",
					EnvVars: []string{"API_ADDRESS"},
				},
				&cli.Uint64SliceFlag{
					Name:    "uints-64",
					EnvVars: []string{"UINTS_64"},
				},
				&cli.StringFlag{
					Name:    "color",
					EnvVars: []string{"COLOR"},
					Value:   "Blue",
				},
			},
			Commands: []*cli.Command{
				{
					Name:  "start",
					Usage: "start service",
					Flags: []cli.Flag{},

					Args:      true,
					ArgsUsage: "[SERVICE [SERVICE]]",
				},
				{
					Name:  "stops",
					Usage: "stop service",
					Flags: []cli.Flag{},

					Args:      true,
					ArgsUsage: "[SERVICE [SERVICE]]",
				},
				{
					Name:  "config",
					Usage: "configure system",
					Flags: []cli.Flag{},
					Subcommands: []*cli.Command{
						{
							Name:  "setoption",
							Usage: "set config option NAME from json VALUE",
							Flags: []cli.Flag{},

							Args:      true,
							ArgsUsage: "NAME VALUE",
						},
					},
				},
			},
			Reader:    os.Stdin,
			Writer:    os.Stdout,
			ErrWriter: os.Stderr,
		},
	})

	type C1 struct {
		*clive.Command
		Bool     bool
		Duration time.Duration
		Float64  float64
		Int64    int64
		Int      int
	}
	type C2 struct {
		*clive.Command
		Ints    []int
		Ints64  []int64
		String  string
		Strings []string
		Uint64  uint64
		Uint    uint
	}
	c1 := &C1{}
	c2 := &C2{}
	tests = append(tests, test{
		[]interface{}{c1, c2},
		&cli.App{
			Name: "tests.test",
			// HelpName: "clive.test",
			Usage: "A new cli application",
			// Version:  "0.0.0",
			Commands: []*cli.Command{
				{
					Name: "c1",
					Flags: []cli.Flag{
						&cli.BoolFlag{Name: "bool", EnvVars: []string{"BOOL"}},
						&cli.DurationFlag{Name: "duration", EnvVars: []string{"DURATION"}},
						&cli.Float64Flag{Name: "float-64", EnvVars: []string{"FLOAT_64"}},
						&cli.Int64Flag{Name: "int-64", EnvVars: []string{"INT_64"}},
						&cli.IntFlag{Name: "int", EnvVars: []string{"INT"}},
					},
				},
				{
					Name: "c2",
					Flags: []cli.Flag{
						&cli.IntSliceFlag{Name: "ints", EnvVars: []string{"INTS"}},
						&cli.Int64SliceFlag{Name: "ints-64", EnvVars: []string{"INTS_64"}},
						&cli.StringFlag{Name: "string", EnvVars: []string{"STRING"}},
						&cli.StringSliceFlag{Name: "strings", EnvVars: []string{"STRINGS"}},
						&cli.Uint64Flag{Name: "uint-64", EnvVars: []string{"UINT_64"}},
						&cli.UintFlag{Name: "uint", EnvVars: []string{"UINT"}},
					},
				},
			},
			Reader:    os.Stdin,
			Writer:    os.Stdout,
			ErrWriter: os.Stderr,
		},
	})

	for ii, tt := range tests {
		t.Run(fmt.Sprint(ii), func(t *testing.T) {
			gotC := clive.Build(tt.objs...)

			// not handled by this library
			gotC.BashComplete = nil

			// can't get this one easily for tests
			gotC.Compiled = time.Time{}

			// function pointers don't compare properly at all for some reason
			gotC.Before = nil
			gotC.Action = nil

			// dont check Metadata (used internally)
			gotC.Metadata = nil

			queue := make([]*cli.Command, len(gotC.Commands))
			copy(queue, gotC.Commands)
			for ; len(queue) != 0; queue = queue[1:] {
				next := queue[0]
				next.Action = nil
				next.Before = nil
				queue = append(queue, next.Subcommands...)
			}

			assert.Equal(t, tt.wantC, gotC)
		})
	}
}

func TestRunDefaults(t *testing.T) {
	type T struct {
		*clive.Command `cli:"usage:'this command does a, b and c'"`
		Run            clive.RunFunc
		One            string `cli:"default:hello"`
		Two            string `cli:"default:there"`
		Three          string `cli:"name:api_address,default:1.2.3.4"`
	}

	gotC := clive.Build(&T{
		Run: func(c *clive.Command, ctx *cli.Context) error {
			flags, ok := c.Current(ctx).(*T)
			assert.True(t, ok)

			assert.Equal(t, flags.One, "hi")
			assert.Equal(t, flags.Two, "world")
			assert.Equal(t, flags.Three, "1.2.3.4")

			return nil
		}})
	err := gotC.Run([]string{"", "--one=hi", "--two=world"})
	if err != nil {
		t.Error(err)
	}
}

func TestRunAll(t *testing.T) {
	v, err := strconv.ParseUint("0xffffffffffffffff", 0, 64)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, uint64(0xffffffffffffffff), v)
	type test struct {
		name string
		obj  interface{}
		args []string
	}
	var tests []test = []test{}

	type Tflags struct {
		*clive.Command
		Run      clive.RunFunc
		Int      int
		Int64    int64
		Uint     uint
		Uint64   uint64
		Float32  float32
		Float64  float64
		Bool     bool
		String   string
		Duration time.Duration
		Ints     []int
		Ints64   []int64
		Uints    []uint
		Uints64  []uint64
		Strings  []string
		Color    ColorT
	}
	// math.MaxUint64

	tests = append(tests, test{
		name: "all flags",
		obj: &Tflags{Run: func(c *clive.Command, ctx *cli.Context) error {
			flags, ok := c.Current(ctx).(*Tflags)
			assert.True(t, ok)

			assert.Equal(t, int(2147483646), flags.Int)
			assert.Equal(t, int64(9123372036854775801), flags.Int64)
			assert.Equal(t, uint(9123372036854775801), flags.Uint)
			assert.Equal(t, uint64(18446744073709551610), flags.Uint64)
			assert.Equal(t, float32(4.5), flags.Float32)
			assert.Equal(t, 4.5, flags.Float64)
			assert.Equal(t, true, flags.Bool)
			assert.Equal(t, "thing", flags.String)
			assert.Equal(t, time.Hour+(time.Minute*5)+(time.Second*10), flags.Duration)
			assert.Equal(t, Red, flags.Color)
			assert.Equal(t, []int{9, 223, 372, 36, 854, 775, 807}, flags.Ints)
			assert.Equal(t, []int64{9123372036854775801, 223, 372, 36, 854, 775, 807}, flags.Ints64)
			assert.Equal(t, []uint{4294967295}, flags.Uints)
			assert.Equal(t, []uint64{18446744073709551615, 0}, flags.Uints64)
			assert.Equal(t, []string{"thing1", "thing2"}, flags.Strings)

			return nil
		}},
		args: []string{
			"",
			"--int=2147483646",
			"--int-64=9123372036854775801",
			"--uint=9123372036854775801",
			"--uint-64=18446744073709551610",
			"--float-32=4.5",
			"--float-64=4.5",
			"--bool=true",
			"--string=thing",
			"--duration=1h5m10s",
			"--color=Red",
			"--ints=9",
			"--ints=223",
			"--ints=372",
			"--ints=36",
			"--ints=854",
			"--ints=775",
			"--ints=807",
			"--uints=4294967295",
			"--uints-64=18446744073709551615",
			"--uints-64=0",
			"--ints-64=9123372036854775801",
			"--ints-64=223",
			"--ints-64=372",
			"--ints-64=36",
			"--ints-64=854",
			"--ints-64=775",
			"--ints-64=807",
			"--strings=thing1",
			"--strings=thing2",
		},
	})

	type Tscalars struct {
		*clive.Command
		Run      clive.RunFunc
		Int      int           `cli:"positional"`
		Int64    int64         `cli:"positional"`
		Uint     uint          `cli:"positional"`
		Uint64   uint64        `cli:"positional"`
		Float32  float32       `cli:"positional"`
		Float64  float64       `cli:"positional"`
		Bool     bool          `cli:"positional"`
		String   string        `cli:"positional"`
		Duration time.Duration `cli:"positional"`
		Color    ColorT        `cli:"positional"`
	}
	tests = append(tests, test{
		name: "scalar args",
		obj: &Tscalars{Run: func(c *clive.Command, ctx *cli.Context) error {
			flags, ok := c.Current(ctx).(*Tscalars)
			assert.True(t, ok)

			assert.Equal(t, int(2147483646), flags.Int)
			assert.Equal(t, int64(9123372036854775801), flags.Int64)
			assert.Equal(t, uint(9123372036854775801), flags.Uint)
			assert.Equal(t, uint64(18446744073709551610), flags.Uint64)
			assert.Equal(t, float32(4.5), flags.Float32)
			assert.Equal(t, 4.5, flags.Float64)
			assert.Equal(t, true, flags.Bool)
			assert.Equal(t, "thing", flags.String)
			assert.Equal(t, time.Hour+(time.Minute*5)+(time.Second*10), flags.Duration)
			assert.Equal(t, Red, flags.Color)
			return nil
		}},
		args: []string{
			"",
			"2147483646",
			"9123372036854775801",
			"9123372036854775801",
			"18446744073709551610",
			"4.5",
			"4.5",
			"true",
			"thing",
			"1h5m10s",
			"Red",
		},
	})
	type (
		TintsArgs struct {
			*clive.Command
			Run  clive.RunFunc
			Args []int `cli:"positional"`
		}
		Tints64Args struct {
			*clive.Command
			Run  clive.RunFunc
			Args []int64 `cli:"positional"`
		}
		TuintsArgs struct {
			*clive.Command
			Run  clive.RunFunc
			Args []uint `cli:"positional"`
		}
		Tuints64Args struct {
			*clive.Command
			Run  clive.RunFunc
			Args []uint64 `cli:"positional"`
		}
		TstringsArgs struct {
			*clive.Command
			Run  clive.RunFunc
			Args []string `cli:"positional"`
		}
	)

	tests = append(tests, test{
		name: "ints variadic",
		obj: &TintsArgs{Run: func(c *clive.Command, ctx *cli.Context) error {
			flags, ok := c.Current(ctx).(*TintsArgs)
			assert.True(t, ok)

			assert.Equal(t, []int{9, 223, 372, 36, 854, 775, 807}, flags.Args)
			return nil
		}},
		args: []string{
			"",
			"9",
			"223",
			"372",
			"36",
			"854",
			"775",
			"807",
		},
	})

	tests = append(tests, test{
		name: "uints variadic",
		obj: &TuintsArgs{Run: func(c *clive.Command, ctx *cli.Context) error {
			flags, ok := c.Current(ctx).(*TuintsArgs)
			assert.True(t, ok)

			assert.Equal(t, []uint{4294967295}, flags.Args)
			return nil
		}},
		args: []string{
			"",
			"4294967295",
		},
	})

	tests = append(tests, test{
		name: "uints64 variadic",
		obj: &Tuints64Args{Run: func(c *clive.Command, ctx *cli.Context) error {
			flags, ok := c.Current(ctx).(*Tuints64Args)
			assert.True(t, ok)

			assert.Equal(t, []uint64{18446744073709551615, 0}, flags.Args)
			return nil
		}},
		args: []string{
			"",
			"18446744073709551615",
			"0",
		},
	})

	tests = append(tests, test{
		name: "ints64 variadic",
		obj: &Tints64Args{Run: func(c *clive.Command, ctx *cli.Context) error {
			flags, ok := c.Current(ctx).(*Tints64Args)
			assert.True(t, ok)

			assert.Equal(t, []int64{9123372036854775801, 223, 372, 36, 854, 775, 807}, flags.Args)
			return nil
		}},
		args: []string{
			"",
			"9123372036854775801",
			"223",
			"372",
			"36",
			"854",
			"775",
			"807",
		},
	})

	tests = append(tests, test{
		name: "strings variadic",
		obj: &TstringsArgs{Run: func(c *clive.Command, ctx *cli.Context) error {
			flags, ok := c.Current(ctx).(*TstringsArgs)
			assert.True(t, ok)

			assert.Equal(t, []string{"thing1", "thing2"}, flags.Args)
			return nil
		}},
		args: []string{
			"",
			"thing1",
			"thing2",
		},
	})

	for _, tv := range tests {
		t.Run(tv.name, func(t *testing.T) {
			gotC := clive.Build(tv.obj)
			err := gotC.Run(tv.args)
			if err != nil {
				t.Error(err)
			}
		})
	}

}

func TestRunAllDefaults(t *testing.T) {
	type T struct {
		*clive.Command
		Run      clive.RunFunc
		Int      int           `cli:"default:2147483646"`
		Int64    int64         `cli:"default:9123372036854775801"`
		Uint     uint          `cli:"default:9123372036854775801"`
		Uint64   uint64        `cli:"default:18446744073709551610"`
		Float32  float32       `cli:"default:4.5"`
		Float64  float64       `cli:"default:4.5"`
		Bool     bool          `cli:"default:true"`
		String   string        `cli:"default:thing"`
		Duration time.Duration `cli:"default:1h5m10s"`
		Ints     []int         `cli:"default:'9,8,7'"`
		Ints64   []int64       `cli:"default:9123372036854775801"`
		Uints    []uint        `cli:"default:4294967295"`
		Uints64  []uint64      `cli:"default:'18446744073709551615,0"`
		Strings  []string      `cli:"default:'thing1,thing2'"`
		Color    ColorT        `cli:"default:'Blue'"`
	}

	gotC := clive.Build(&T{Run: func(c *clive.Command, ctx *cli.Context) error {
		flags, ok := c.Current(ctx).(*T)
		assert.True(t, ok)

		assert.Equal(t, int(2147483646), flags.Int)
		assert.Equal(t, int64(9123372036854775801), flags.Int64)
		assert.Equal(t, uint(9123372036854775801), flags.Uint)
		assert.Equal(t, uint64(18446744073709551610), flags.Uint64)
		assert.Equal(t, float32(4.5), flags.Float32)
		assert.Equal(t, 4.5, flags.Float64)
		assert.Equal(t, true, flags.Bool)
		assert.Equal(t, "thing", flags.String)
		assert.Equal(t, time.Hour+(time.Minute*5)+(time.Second*10), flags.Duration)
		assert.Equal(t, []int{9, 8, 7}, flags.Ints)
		assert.Equal(t, []int64{9123372036854775801}, flags.Ints64)
		assert.Equal(t, []uint{4294967295}, flags.Uints)
		assert.Equal(t, []uint64{18446744073709551615, 0}, flags.Uints64)
		assert.Equal(t, []string{"thing1", "thing2"}, flags.Strings)
		assert.Equal(t, Blue, flags.Color)

		return nil
	}})
	err := gotC.Run([]string{
		"",
	})
	if err != nil {
		t.Error(err)
	}
}
