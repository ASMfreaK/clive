package clive

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
)

func TestBuild(t *testing.T) {
	type test struct {
		objs  []interface{}
		wantC *cli.App
	}
	tests := []test{}

	stub := func(*cli.Context) error { return nil }

	type T struct {
		cli.Command               `cli:"usage:'this command does a, b and c'"`
		FlagPostgresDsn           string `cli:"default:hello"`
		FlagProcessSchedule       string `cli:"hidden:true"`
		FlagApplicationAPIAddress string `cli:"name:api_address"`
	}
	tests = append(tests, test{
		[]interface{}{
			T{Command: cli.Command{Action: stub}},
		},
		&cli.App{
			Name: "clive.test",
			// HelpName: "clive.test",
			Usage: "this command does a, b and c",
			// Version:  "0.0.0",
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
			},
			Reader:    os.Stdin,
			Writer:    os.Stdout,
			ErrWriter: os.Stderr,
		},
	})

	type C1 struct {
		cli.Command
		FlagBool     bool
		FlagDuration time.Duration
		FlagFloat64  float64
		FlagInt64    int64
		FlagInt      int
	}
	type C2 struct {
		cli.Command
		FlagInts    []int
		FlagInts64  []int64
		FlagString  string
		FlagStrings []string
		FlagUint64  uint64
		FlagUint    uint
	}
	tests = append(tests, test{
		[]interface{}{
			C1{Command: cli.Command{Action: stub}},
			C2{Command: cli.Command{Action: stub}},
		},
		&cli.App{
			Name: "clive.test",
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
			gotC, err := build(tt.objs...)
			if err != nil {
				t.Error(err)
			}

			// not handled by this library
			gotC.BashComplete = nil

			// can't get this one easily for tests
			gotC.Compiled = time.Time{}

			// function pointers don't compare properly at all for some reason
			gotC.Action = nil

			for i := range gotC.Commands {
				gotC.Commands[i].Action = nil
			}

			assert.Equal(t, tt.wantC, gotC)
		})
	}
}

func TestBuildSubcommands(t *testing.T) {
	type test struct {
		objs  []interface{}
		wantC []*cli.Command
	}
	tests := []test{}

	stub := func(*cli.Context) error { return nil }

	type C1 struct {
		cli.Command
		FlagBool     bool
		FlagDuration time.Duration
		FlagFloat64  float64
		FlagInt64    int64
		FlagInt      int
	}
	type C2 struct {
		cli.Command
		FlagInts    []int
		FlagInts64  []int64
		FlagString  string
		FlagStrings []string
		FlagUint64  uint64
		FlagUint    uint
	}
	tests = append(tests, test{
		[]interface{}{
			C1{Command: cli.Command{Action: stub}},
			C2{Command: cli.Command{Action: stub}},
		},
		[]*cli.Command{
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
	})

	for ii, tt := range tests {
		t.Run(fmt.Sprint(ii), func(t *testing.T) {
			gotC, err := buildCommands(tt.objs...)
			if err != nil {
				t.Error(err)
			}

			for i := range gotC {
				gotC[i].Action = nil
			}

			assert.Equal(t, tt.wantC, gotC)
		})
	}

}

func TestRunDefaults(t *testing.T) {
	type T struct {
		cli.Command `cli:"usage:'this command does a, b and c'"`
		FlagOne     string `cli:"default:hello"`
		FlagTwo     string `cli:"default:there"`
		FlagThree   string `cli:"name:api_address,default:1.2.3.4"`
	}

	gotC, err := build(T{Command: cli.Command{Action: func(c *cli.Context) error {
		flags, ok := Flags(T{}, c).(T)
		assert.True(t, ok)

		assert.Equal(t, flags.FlagOne, "hi")
		assert.Equal(t, flags.FlagTwo, "world")
		assert.Equal(t, flags.FlagThree, "1.2.3.4")

		return nil
	}}})
	if err != nil {
		t.Error(err)
	}
	err = gotC.Run([]string{"", "--one=hi", "--two=world"})
	if err != nil {
		t.Error(err)
	}
}

func TestRunAll(t *testing.T) {
	type T struct {
		cli.Command
		FlagInt      int
		FlagInt64    int64
		FlagUint     uint
		FlagUint64   uint64
		FlagFloat32  float32
		FlagFloat64  float64
		FlagBool     bool
		FlagString   string
		FlagDuration time.Duration
		FlagInts     []int
		FlagInts64   []int64
		FlagStrings  []string
	}

	gotC := Build(T{Command: cli.Command{Action: func(c *cli.Context) error {
		flags, ok := Flags(T{}, c).(T)
		assert.True(t, ok)

		assert.Equal(t, int(2147483646), flags.FlagInt)
		assert.Equal(t, int64(9123372036854775801), flags.FlagInt64)
		assert.Equal(t, uint(9123372036854775801), flags.FlagUint)
		assert.Equal(t, uint64(18446744073709551610), flags.FlagUint64)
		assert.Equal(t, float32(4.5), flags.FlagFloat32)
		assert.Equal(t, 4.5, flags.FlagFloat64)
		assert.Equal(t, true, flags.FlagBool)
		assert.Equal(t, "thing", flags.FlagString)
		assert.Equal(t, time.Hour+(time.Minute*5)+(time.Second*10), flags.FlagDuration)
		assert.Equal(t, []int{9, 223, 372, 36, 854, 775, 807}, flags.FlagInts)
		assert.Equal(t, []int64{9123372036854775801, 223, 372, 36, 854, 775, 807}, flags.FlagInts64)
		assert.Equal(t, []string{"thing1", "thing2"}, flags.FlagStrings)

		return nil
	}}})
	err := gotC.Run([]string{
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
		"--ints=9",
		"--ints=223",
		"--ints=372",
		"--ints=36",
		"--ints=854",
		"--ints=775",
		"--ints=807",
		"--ints-64=9123372036854775801",
		"--ints-64=223",
		"--ints-64=372",
		"--ints-64=36",
		"--ints-64=854",
		"--ints-64=775",
		"--ints-64=807",
		"--strings=thing1",
		"--strings=thing2",
	})
	if err != nil {
		t.Error(err)
	}
}

func TestRunAllDefaults(t *testing.T) {
	type T struct {
		cli.Command
		FlagInt      int           `cli:"default:2147483646"`
		FlagInt64    int64         `cli:"default:9123372036854775801"`
		FlagUint     uint          `cli:"default:9123372036854775801"`
		FlagUint64   uint64        `cli:"default:18446744073709551610"`
		FlagFloat32  float32       `cli:"default:4.5"`
		FlagFloat64  float64       `cli:"default:4.5"`
		FlagBool     bool          `cli:"default:true"`
		FlagString   string        `cli:"default:thing"`
		FlagDuration time.Duration `cli:"default:1h5m10s"`
		FlagInts     []int         `cli:"default:'9,8,7'"`
		FlagInts64   []int64       `cli:"default:9123372036854775801"`
		FlagStrings  []string      `cli:"default:'thing1,thing2'"`
	}

	gotC, err := build(T{Command: cli.Command{Action: func(c *cli.Context) error {
		flags, ok := Flags(T{}, c).(T)
		assert.True(t, ok)

		assert.Equal(t, int(2147483646), flags.FlagInt)
		assert.Equal(t, int64(9123372036854775801), flags.FlagInt64)
		assert.Equal(t, uint(9123372036854775801), flags.FlagUint)
		assert.Equal(t, uint64(18446744073709551610), flags.FlagUint64)
		assert.Equal(t, float32(4.5), flags.FlagFloat32)
		assert.Equal(t, 4.5, flags.FlagFloat64)
		assert.Equal(t, true, flags.FlagBool)
		assert.Equal(t, "thing", flags.FlagString)
		assert.Equal(t, time.Hour+(time.Minute*5)+(time.Second*10), flags.FlagDuration)
		assert.Equal(t, []int{9, 8, 7}, flags.FlagInts)
		assert.Equal(t, []int64{9123372036854775801}, flags.FlagInts64)
		assert.Equal(t, []string{"thing1", "thing2"}, flags.FlagStrings)

		return nil
	}}})
	if err != nil {
		t.Error(err)
	}
	err = gotC.Run([]string{
		"",
	})
	if err != nil {
		t.Error(err)
	}
}
