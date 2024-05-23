package clive2_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	clive "github.com/ASMfreaK/clive2"
	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
)

// func TestBits(t *testing.T) {
// 	r, err := clive.ParseAny[ColorT]("Red")
// 	assert.NoError(t, err)
// 	assert.Equal(t, Red, r)

// 	rs, err := clive.ParseAny[[]ColorT]("Red,Green")
// 	assert.NoError(t, err)
// 	assert.Equal(t, []ColorT{Red, Green}, rs)
// }

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

func (c *ColorT) Variants() []string {
	return colorStrings
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

	Run clive.RunFunc

	Service []string `cli:"positional,usage:'services to use',required:false"`
}

func (start *Start) Action(*cli.Context) error {
	fmt.Printf("start %+v\n", start)
	return nil
}

type Stop struct {
	*clive.Command `cli:"name:'stops',usage:'stop service'"`

	Run clive.RunFunc

	Service []string `cli:"positional,usage:'services to use',required:false"`
}

func (*Stop) Description() string {
	return `
this is a long and very and descriptive text
describing stop command
`
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
func (c *Role) Variants() []string {
	return roleStrings
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

	Run clive.RunFunc

	Name  string `cli:"positional"`
	Value Json   `cli:"positional"`
}

type ComposedOption struct {
	Role Role
	Port int
}

type StartStop struct {
	*Start
	*Stop
}

type App struct {
	*clive.Command `cli:"name:'testCli',usage:'test command'"`

	Subcommands struct {
		StartStop
		Config *struct {
			*clive.Command `cli:"name:'config',usage:'configure system'"`

			Subcommands struct {
				*SetOption
			}

			ExpensiveFieldToCompute *int `cli:"-"`
		}
	}

	PostgresDsn           string  `cli:"default:hello"`
	ProcessSchedule       string  `cli:"hidden:true"`
	ApplicationAPIAddress *string `cli:"name:api_address,alias:'a,i'"`
	Uints64               []uint64

	Color  ColorT         `cli:"default:Blue,alias:'c'"`
	Input  ComposedOption `cli:"inline"`
	Output ComposedOption `cli:"inline"`
}

func (*App) Version() string {
	return "some-version-string"
}

// func TestQueue(t *testing.T) {
// 	rapid.Check(t, func(t *rapid.T) {
// 		type rapidType struct {
// 			refType reflect.Type

// 		}
// 		types := []reflect.Type{
// 			clive.Reflected[int](),
// 			clive.Reflected[int64](),
// 			clive.Reflected[uint](),
// 			clive.Reflected[uint64](),
// 			clive.Reflected[float32](),
// 			clive.Reflected[float64](),
// 			clive.Reflected[bool](),
// 			clive.Reflected[string](),
// 			clive.Reflected[time.Duration](),
// 			clive.Reflected[ColorT](),
// 			clive.Reflected[[]int](),
// 			clive.Reflected[[]int64](),
// 			clive.Reflected[[]uint](),
// 			clive.Reflected[[]uint64](),
// 			clive.Reflected[[]float32](),
// 			clive.Reflected[[]float64](),
// 			clive.Reflected[[]bool](),
// 			clive.Reflected[[]string](),
// 			clive.Reflected[[]time.Duration](),
// 			clive.Reflected[[]ColorT](),
// 		}
// 		typesGen := rapid.SampledFrom(types)
// 	})
// }

func TestBuild(t *testing.T) {
	type test struct {
		obj   interface{}
		wantC *cli.App

		panicErr error
	}
	tests := []test{
		{
			panicErr: clive.ErrNil,

			obj: nil,
		},
		{
			panicErr: &clive.ByValueError{Type: ""},

			obj: struct{}{},
		},
		{
			panicErr: &clive.WrongFirstFieldError{NumFields: 0, FieldName: "", Type: ""},

			obj: &struct{}{},
		},
		{
			panicErr: &clive.WrongFirstFieldError{NumFields: 1, FieldName: "yay", Type: "int"},

			obj: &struct {
				yay int
			}{},
		},

		{
			panicErr: &clive.PositionalAfterVariadicError{CurrentName: "pos-2", FirstName: "pos-1"},

			obj: &struct {
				*clive.Command
				pos1 []string `cli:"positional"`
				pos2 []string `cli:"positional"`
			}{},
		},
	}

	// stub := func(*cli.Context) error { return nil }

	app := &App{}
	tests = append(tests, test{
		obj: app,
		wantC: &cli.App{
			Name: "tests.test",
			// HelpName: "clive.test",
			Usage:           "test command",
			Version:         "some-version-string",
			HideHelpCommand: true,
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
					Aliases: []string{"a", "i"},
				},
				&cli.Uint64SliceFlag{
					Name:    "uints-64",
					EnvVars: []string{"UINTS_64"},
				},
				&cli.StringFlag{
					Name:    "color",
					EnvVars: []string{"COLOR"},
					Value:   "Blue",
					Aliases: []string{"c"},
					Usage:   "possible values: [Red, Green, Blue]",
				},

				&cli.StringFlag{
					Name:    "input-role",
					EnvVars: []string{"INPUT_ROLE"},
					Usage:   "possible values: [server, client]",
				},
				&cli.IntFlag{
					Name:    "input-port",
					EnvVars: []string{"INPUT_PORT"},
				},
				&cli.StringFlag{
					Name:    "output-role",
					EnvVars: []string{"OUTPUT_ROLE"},
					Usage:   "possible values: [server, client]",
				},
				&cli.IntFlag{
					Name:    "output-port",
					EnvVars: []string{"OUTPUT_PORT"},
				},
			},
			Commands: []*cli.Command{
				{
					Name:  "start",
					Usage: "start service",
					Flags: []cli.Flag{},

					Args:      true,
					ArgsUsage: "[SERVICE [SERVICE]]",

					HideHelpCommand: true,
				},
				{
					Name:        "stops",
					Usage:       "stop service",
					Description: (*Stop)(nil).Description(),
					Flags:       []cli.Flag{},

					Args:      true,
					ArgsUsage: "[SERVICE [SERVICE]]",

					HideHelpCommand: true,
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

							HideHelpCommand: true,
						},
					},
					HideHelpCommand: true,
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
		Run     clive.RunFunc
		Ints    []int
		Ints64  []int64
		String  string
		Strings []string
		Uint64  uint64
		Uint    uint
	}

	type C12 struct {
		*clive.Command
		Subcommands struct {
			*C1
			*C2
		}
	}
	tests = append(tests, test{
		obj: &C12{},
		wantC: &cli.App{
			Name: "tests.test",
			// HelpName: "clive.test",
			Usage: "",
			// Version:  "0.0.0",
			HideHelpCommand: true,
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
					HideHelpCommand: true,
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
					HideHelpCommand: true,
				},
			},
			Flags:     []cli.Flag{},
			Reader:    os.Stdin,
			Writer:    os.Stdout,
			ErrWriter: os.Stderr,
		},
	})

	countObj := &struct {
		*clive.Command
		Silent clive.Counter `cli:"alias:s"`
	}{}
	tests = append(tests, test{
		obj: countObj,
		wantC: &cli.App{
			Name: "tests.test",
			// HelpName: "clive.test",
			Usage: "",
			// Version:  "0.0.0",
			HideHelpCommand: true,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:    "silent",
					Aliases: []string{"s"},
					EnvVars: []string{"SILENT"},
					Count:   &countObj.Silent.Value,
				},
			},
			Reader:    os.Stdin,
			Writer:    os.Stdout,
			ErrWriter: os.Stderr,
		},
	})

	for ii, tt := range tests {
		t.Run(fmt.Sprint(ii), func(t *testing.T) {
			var gotC *cli.App
			if tt.panicErr == nil {
				gotC = clive.Build(tt.obj)
			} else {
				assert.PanicsWithError(t, tt.panicErr.Error(), func() { gotC = clive.Build(tt.obj) }, "this test should panic")
				return
			}

			// not handled by this library
			gotC.BashComplete = nil

			// can't get this one easily for tests
			gotC.Compiled = time.Time{}

			// function pointers don't compare properly at all for some reason
			gotC.Before = nil
			gotC.Action = nil
			gotC.After = nil

			// dont check Metadata (used internally)
			gotC.Metadata = nil

			queue := make([]*cli.Command, len(gotC.Commands))
			copy(queue, gotC.Commands)
			for ; len(queue) != 0; queue = queue[1:] {
				next := queue[0]
				next.Action = nil
				next.Before = nil
				next.After = nil
				queue = append(queue, next.Subcommands...)
			}

			ok := assert.Equal(t, tt.wantC, gotC)

			if !ok && len(os.Getenv("TEST_BUILD_SHOW_FULL_TEXT")) > 0 {
				b := &bytes.Buffer{}
				spew.Fdump(b, tt.wantC)
				s := bufio.NewScanner(b)
				i := 0
				for s.Scan() {
					i += 1
					fmt.Printf("%03d: %s\n", i, s.Bytes())
				}
			}
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
		Color    ColorT
		Input    ComposedOption `cli:"inline"`
		Output   ComposedOption `cli:"inline"`
		Ints     []int
		Ints64   []int64
		Uints    []uint
		Uints64  []uint64
		Strings  []string
		Colors   []ColorT
	}

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
			assert.Equal(t, Server, flags.Input.Role)
			assert.Equal(t, 81234, flags.Input.Port)
			assert.Equal(t, Client, flags.Output.Role)
			assert.Equal(t, 81235, flags.Output.Port)
			assert.Equal(t, []int{9, 223, 372, 36, 854, 775, 807}, flags.Ints)
			assert.Equal(t, []int64{9123372036854775801, 223, 372, 36, 854, 775, 807}, flags.Ints64)
			assert.Equal(t, []uint{4294967295}, flags.Uints)
			assert.Equal(t, []uint64{18446744073709551615, 0}, flags.Uints64)
			assert.Equal(t, []string{"thing1", "thing2"}, flags.Strings)
			assert.Equal(t, []ColorT{Red, Green, Red, Blue, Red}, flags.Colors)

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
			"--input-role=server",
			"--input-port=81234",
			"--output-role=client",
			"--output-port=81235",
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
			"--colors=Red",
			"--colors=Green",
			"--colors=Red",
			"--colors=Blue",
			"--colors=Red",
		},
	})

	type TflagsPtr struct {
		*clive.Command
		Run      clive.RunFunc
		Int      *int
		Int64    *int64
		Uint     *uint
		Uint64   *uint64
		Float32  *float32
		Float64  *float64
		Bool     *bool
		String   *string
		Duration *time.Duration
		Ints     *[]int
		Ints64   *[]int64
		Uints    *[]uint
		Uints64  *[]uint64
		Strings  *[]string
		Color    *ColorT

		UnspecifiedInt *int
	}
	tests = append(tests, test{
		name: "all flags",
		obj: &TflagsPtr{Run: func(c *clive.Command, ctx *cli.Context) error {
			flags, ok := c.Current(ctx).(*TflagsPtr)
			assert.True(t, ok)

			assert.Equal(t, (*int)(nil), flags.UnspecifiedInt)

			assert.NotEqual(t, (*int)(nil), flags.Int)
			assert.NotEqual(t, (*int64)(nil), flags.Int64)
			assert.NotEqual(t, (*uint)(nil), flags.Uint)
			assert.NotEqual(t, (*uint64)(nil), flags.Uint64)
			assert.NotEqual(t, (*float32)(nil), flags.Float32)
			assert.NotEqual(t, (*float32)(nil), flags.Float64)
			assert.NotEqual(t, (*bool)(nil), flags.Bool)
			assert.NotEqual(t, (*string)(nil), flags.String)
			assert.NotEqual(t, (*time.Duration)(nil), flags.Duration)
			assert.NotEqual(t, (*ColorT)(nil), flags.Color)
			assert.NotEqual(t, (*[]int)(nil), flags.Ints)
			assert.NotEqual(t, (*[]int64)(nil), flags.Ints64)
			assert.NotEqual(t, (*[]uint)(nil), flags.Uints)
			assert.NotEqual(t, (*[]uint64)(nil), flags.Uints64)
			assert.NotEqual(t, (*[]string)(nil), flags.Strings)

			assert.Equal(t, int(2147483646), *flags.Int)
			assert.Equal(t, int64(9123372036854775801), *flags.Int64)
			assert.Equal(t, uint(9123372036854775801), *flags.Uint)
			assert.Equal(t, uint64(18446744073709551610), *flags.Uint64)
			assert.Equal(t, float32(4.5), *flags.Float32)
			assert.Equal(t, 4.5, *flags.Float64)
			assert.Equal(t, true, *flags.Bool)
			assert.Equal(t, "thing", *flags.String)
			assert.Equal(t, time.Hour+(time.Minute*5)+(time.Second*10), *flags.Duration)
			assert.Equal(t, Red, *flags.Color)
			assert.Equal(t, []int{9, 223, 372, 36, 854, 775, 807}, *flags.Ints)
			assert.Equal(t, []int64{9123372036854775801, 223, 372, 36, 854, 775, 807}, *flags.Ints64)
			assert.Equal(t, []uint{4294967295}, *flags.Uints)
			assert.Equal(t, []uint64{18446744073709551615, 0}, *flags.Uints64)
			assert.Equal(t, []string{"thing1", "thing2"}, *flags.Strings)

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

	type TscalarsPtr struct {
		*clive.Command
		Run      clive.RunFunc
		Int      *int           `cli:"positional"`
		Int64    *int64         `cli:"positional"`
		Uint     *uint          `cli:"positional"`
		Uint64   *uint64        `cli:"positional"`
		Float32  *float32       `cli:"positional"`
		Float64  *float64       `cli:"positional"`
		Bool     *bool          `cli:"positional"`
		String   *string        `cli:"positional"`
		Duration *time.Duration `cli:"positional"`
		Color    *ColorT        `cli:"positional"`
	}
	tests = append(tests, test{
		name: "scalar args",
		obj: &TscalarsPtr{Run: func(c *clive.Command, ctx *cli.Context) error {
			flags, ok := c.Current(ctx).(*TscalarsPtr)
			assert.True(t, ok)

			assert.NotEqual(t, (*int)(nil), flags.Int)
			assert.NotEqual(t, (*int64)(nil), flags.Int64)
			assert.NotEqual(t, (*uint)(nil), flags.Uint)
			assert.NotEqual(t, (*uint64)(nil), flags.Uint64)
			assert.NotEqual(t, (*float32)(nil), flags.Float32)
			assert.NotEqual(t, (*float32)(nil), flags.Float64)
			assert.NotEqual(t, (*bool)(nil), flags.Bool)
			assert.NotEqual(t, (*string)(nil), flags.String)
			assert.NotEqual(t, (*time.Duration)(nil), flags.Duration)
			assert.NotEqual(t, (*ColorT)(nil), flags.Color)

			assert.Equal(t, int(2147483646), *flags.Int)
			assert.Equal(t, int64(9123372036854775801), *flags.Int64)
			assert.Equal(t, uint(9123372036854775801), *flags.Uint)
			assert.Equal(t, uint64(18446744073709551610), *flags.Uint64)
			assert.Equal(t, float32(4.5), *flags.Float32)
			assert.Equal(t, 4.5, *flags.Float64)
			assert.Equal(t, true, *flags.Bool)
			assert.Equal(t, "thing", *flags.String)
			assert.Equal(t, time.Hour+(time.Minute*5)+(time.Second*10), *flags.Duration)
			assert.Equal(t, Red, *flags.Color)
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
		TIntsArgs struct {
			*clive.Command
			Run  clive.RunFunc
			Args []int `cli:"positional"`
		}
		TIntsArgsPtr struct {
			*clive.Command
			Run  clive.RunFunc
			Args *[]int `cli:"positional"`
		}
		TInts64Args struct {
			*clive.Command
			Run  clive.RunFunc
			Args []int64 `cli:"positional"`
		}
		TInts64ArgsPtr struct {
			*clive.Command
			Run  clive.RunFunc
			Args *[]int64 `cli:"positional"`
		}
		TuintsArgs struct {
			*clive.Command
			Run  clive.RunFunc
			Args []uint `cli:"positional"`
		}
		TuintsArgsPtr struct {
			*clive.Command
			Run  clive.RunFunc
			Args *[]uint `cli:"positional"`
		}
		Tuints64Args struct {
			*clive.Command
			Run  clive.RunFunc
			Args []uint64 `cli:"positional"`
		}
		Tuints64ArgsPtr struct {
			*clive.Command
			Run  clive.RunFunc
			Args *[]uint64 `cli:"positional"`
		}
		TstringsArgs struct {
			*clive.Command
			Run  clive.RunFunc
			Args []string `cli:"positional"`
		}
		TstringsArgsPtr struct {
			*clive.Command
			Run  clive.RunFunc
			Args *[]string `cli:"positional"`
		}
		TColorsArgs struct {
			*clive.Command
			Run  clive.RunFunc
			Args []ColorT `cli:"positional"`
		}
		TColorsArgsPtr struct {
			*clive.Command
			Run  clive.RunFunc
			Args *[]ColorT `cli:"positional"`
		}
	)

	tests = append(tests, test{
		name: "ints variadic",
		obj: &TIntsArgs{Run: func(c *clive.Command, ctx *cli.Context) error {
			flags, ok := c.Current(ctx).(*TIntsArgs)
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
		obj: &TInts64Args{Run: func(c *clive.Command, ctx *cli.Context) error {
			flags, ok := c.Current(ctx).(*TInts64Args)
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

	tests = append(tests, test{
		name: "Colors variadic",
		obj: &TColorsArgs{Run: func(c *clive.Command, ctx *cli.Context) error {
			flags, ok := c.Current(ctx).(*TColorsArgs)
			assert.True(t, ok)

			assert.Equal(t, []ColorT{Red, Green, Blue}, flags.Args)
			return nil
		}},
		args: []string{
			"",
			"Red",
			"Green",
			"Blue",
		},
	})

	tests = append(tests, test{
		name: "ints pointer variadic",
		obj: &TIntsArgsPtr{Run: func(c *clive.Command, ctx *cli.Context) error {
			flags, ok := c.Current(ctx).(*TIntsArgsPtr)
			assert.True(t, ok)
			assert.NotEqual(t, (*[]int)(nil), flags.Args)

			assert.Equal(t, []int{9, 223, 372, 36, 854, 775, 807}, *flags.Args)
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
		name: "uints pointer variadic",
		obj: &TuintsArgsPtr{Run: func(c *clive.Command, ctx *cli.Context) error {
			flags, ok := c.Current(ctx).(*TuintsArgsPtr)
			assert.True(t, ok)
			assert.NotEqual(t, (*[]uint)(nil), flags.Args)

			assert.Equal(t, []uint{4294967295}, *flags.Args)
			return nil
		}},
		args: []string{
			"",
			"4294967295",
		},
	})

	tests = append(tests, test{
		name: "uints64 pointer variadic",
		obj: &Tuints64ArgsPtr{Run: func(c *clive.Command, ctx *cli.Context) error {
			flags, ok := c.Current(ctx).(*Tuints64ArgsPtr)
			assert.True(t, ok)
			assert.NotEqual(t, (*[]uint64)(nil), flags.Args)

			assert.Equal(t, []uint64{18446744073709551615, 0}, *flags.Args)
			return nil
		}},
		args: []string{
			"",
			"18446744073709551615",
			"0",
		},
	})

	tests = append(tests, test{
		name: "ints64 pointer variadic",
		obj: &TInts64ArgsPtr{Run: func(c *clive.Command, ctx *cli.Context) error {
			flags, ok := c.Current(ctx).(*TInts64ArgsPtr)
			assert.True(t, ok)
			assert.NotEqual(t, (*[]int64)(nil), flags.Args)

			assert.Equal(t, []int64{9123372036854775801, 223, 372, 36, 854, 775, 807}, *flags.Args)
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
		name: "strings pointer variadic",
		obj: &TstringsArgsPtr{Run: func(c *clive.Command, ctx *cli.Context) error {
			flags, ok := c.Current(ctx).(*TstringsArgsPtr)
			assert.True(t, ok)
			assert.NotEqual(t, (*[]string)(nil), flags.Args)

			assert.Equal(t, []string{"thing1", "thing2"}, *flags.Args)
			return nil
		}},
		args: []string{
			"",
			"thing1",
			"thing2",
		},
	})

	tests = append(tests, test{
		name: "Colors pointer variadic",
		obj: &TColorsArgsPtr{Run: func(c *clive.Command, ctx *cli.Context) error {
			flags, ok := c.Current(ctx).(*TColorsArgsPtr)
			assert.True(t, ok)
			assert.NotEqual(t, (*[]ColorT)(nil), flags.Args)

			assert.Equal(t, []ColorT{Red, Green, Blue}, *flags.Args)
			return nil
		}},
		args: []string{
			"",
			"Red",
			"Green",
			"Blue",
		},
	})

	type Tcounter struct {
		*clive.Command `cli:"shortOpt"`

		Run    clive.RunFunc
		Silent clive.Counter `cli:"alias:s"`
	}

	tests = append(tests, test{
		name: "counter none",
		obj: &Tcounter{Run: func(c *clive.Command, ctx *cli.Context) error {
			flags, ok := c.Current(ctx).(*Tcounter)
			assert.True(t, ok)
			assert.Equal(t, 0, flags.Silent.Value)
			return nil
		}},
		args: []string{
			"",
		},
	})

	tests = append(tests, test{
		name: "counter some",
		obj: &Tcounter{Run: func(c *clive.Command, ctx *cli.Context) error {
			flags, ok := c.Current(ctx).(*Tcounter)
			assert.True(t, ok)
			assert.Equal(t, 2, flags.Silent.Value)
			return nil
		}},
		args: []string{
			"",
			"--silent", "--silent",
		},
	})

	tests = append(tests, test{
		name: "counter some",
		obj: &Tcounter{Run: func(c *clive.Command, ctx *cli.Context) error {
			flags, ok := c.Current(ctx).(*Tcounter)
			assert.True(t, ok)
			assert.Equal(t, 3, flags.Silent.Value)
			return nil
		}},
		args: []string{
			"",
			"-sss",
		},
	})

	// ========================================================
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
	type test struct {
		name string
		obj  interface{}
	}
	var tests []test = []test{}

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
		Color    ColorT        `cli:"default:'Blue'"`
		Ints     []int         `cli:"default:'9,8,7'"`
		Ints64   []int64       `cli:"default:9123372036854775801"`
		Uints    []uint        `cli:"default:4294967295"`
		Uints64  []uint64      `cli:"default:'18446744073709551615,0"`
		Strings  []string      `cli:"default:'thing1,thing2'"`
	}

	tests = append(tests, test{
		name: "default flags",
		obj: &T{Run: func(c *clive.Command, ctx *cli.Context) error {
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
		}},
	})

	for _, tv := range tests {
		t.Run(tv.name, func(t *testing.T) {
			gotC := clive.Build(tv.obj)
			err := gotC.Run([]string{
				"",
			})
			if err != nil {
				t.Error(err)
			}
		})
	}
}
