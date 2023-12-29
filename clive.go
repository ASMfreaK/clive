package clive

import (
	"encoding"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

type Actionable interface {
	Action(*cli.Context) error
}

// Build constructs a urfave/cli App from an instance of a decorated struct
// Since it is designed to be used 1. on initialisation and; 2. with static data
// that is compile-time only - it does not return an error but instead panics.
// The idea is you will do all your setup once and as long as it doesn't change
// this will never break, so there is little need to pass errors back.
func Build(objs ...interface{}) (c *cli.App) {
	c, err := build(objs...)
	if err != nil {
		panic(err)
	}
	return
}

// BuildSubcommands constructs an array of urfave/cli Commands to be used as subcommands
// Since it is designed to be used 1. on initialisation and; 2. with static data
// that is compile-time only - it does not return an error but instead panics.
// The idea is you will do all your setup once and as long as it doesn't change
// this will never break, so there is little need to pass errors back.
func BuildSubcommands(objs ...interface{}) (commands []*cli.Command) {
	commands, err := buildCommands(objs...)
	if err != nil {
		panic(err)
	}
	return
}

func getFlagName(flag cli.Flag) string {
	return flag.Names()[0]
}

// Flags is a helper function for use within a command Action function. It takes
// an instance of the struct that was used to generate the command and the
// cli.Context pointer that is passed to the action function. It will then
// call the necessary flag access functions (such as c.String("...")) and return
// an instance of the input struct with the necessary fields set.
func Flags(obj interface{}, c *cli.Context) (result interface{}) {
	if obj == nil {
		panic("obj is null")
	}

	objValue := reflect.ValueOf(obj)
	for objValue.Kind() == reflect.Ptr {
		objValue = objValue.Elem()
	}

	objType := objValue.Type()

	resultValue := reflect.New(objType).Elem()

	flagsForValue(&resultValue, objType, c)

	return resultValue.Interface()
}

func flagsForActionable(act Actionable, c *cli.Context) Actionable {

	objValue := reflect.ValueOf(act)
	for objValue.Kind() == reflect.Ptr {
		objValue = objValue.Elem()
	}

	objType := objValue.Type()

	flagsForValue(&objValue, objType, c)

	return act
}

func flagsForValue(obj *reflect.Value, objType reflect.Type, c *cli.Context) {
	for i := 0; i < objType.NumField(); i++ {
		fieldType := objType.Field(i)
		cmdmeta, err := parseMeta(fieldType.Tag.Get("cli"))
		if err != nil {
			panic(err)
		}
		if cmdmeta.Skipped {
			continue
		}

		if strings.HasPrefix(fieldType.Name, "Flag") {
			flag, err := flagFromType(fieldType, cmdmeta)
			if err != nil {
				panic(errors.Wrap(err, "failed to generate flag from struct field"))
			}

			switch fieldType.Type.String() {
			case "int":
				obj.FieldByName(fieldType.Name).SetInt(int64(c.Int(getFlagName(flag))))
			case "int64":
				obj.FieldByName(fieldType.Name).SetInt(c.Int64(getFlagName(flag)))
			case "uint":
				obj.FieldByName(fieldType.Name).SetUint(uint64(c.Uint(getFlagName(flag))))
			case "uint64":
				obj.FieldByName(fieldType.Name).SetUint(c.Uint64(getFlagName(flag)))
			case "float32":
				obj.FieldByName(fieldType.Name).SetFloat(c.Float64(getFlagName(flag)))
			case "float64":
				obj.FieldByName(fieldType.Name).SetFloat(c.Float64(getFlagName(flag)))
			case "bool":
				obj.FieldByName(fieldType.Name).SetBool(c.Bool(getFlagName(flag)))
			case "string":
				obj.FieldByName(fieldType.Name).SetString(c.String(getFlagName(flag)))
			case "time.Duration":
				obj.FieldByName(fieldType.Name).SetInt(c.Duration(getFlagName(flag)).Nanoseconds())
			case "[]int":
				obj.FieldByName(fieldType.Name).Set(genericSliceOf(c.IntSlice(getFlagName(flag))))
			case "[]int64":
				obj.FieldByName(fieldType.Name).Set(genericSliceOf(c.Int64Slice(getFlagName(flag))))
			// case "[]uint":
			// 	obj.FieldByName(fieldType.Name).Set(genericSliceOf(c.IntSlice(getFlagName(flag))))
			// case "[]uint64":
			// 	obj.FieldByName(fieldType.Name).Set(genericSliceOf(c.Int64Slice(getFlagName(flag))))
			case "[]string":
				obj.FieldByName(fieldType.Name).Set(genericSliceOf(c.StringSlice(getFlagName(flag))))
			default:
				if reflect.PointerTo(fieldType.Type).Implements(TextUnmarshalerType) {
					obj.FieldByName(fieldType.Name).Addr().Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(c.String(getFlagName(flag))))
				} else {
					panic("unsupported type")
				}
			}
		}
	}
}

// given a generic slice type, returns a reflected version of that slice with
// all elements inserted.
func genericSliceOf(slice interface{}) reflect.Value {
	sliceValue := reflect.ValueOf(slice)
	length := sliceValue.Len()
	sliceAddr := reflect.New(reflect.MakeSlice(
		reflect.TypeOf(slice),
		length,
		length,
	).Type())
	for i := 0; i < length; i++ {
		value := sliceValue.Index(i)
		ap := reflect.Append(sliceAddr.Elem(), value)
		sliceAddr.Elem().Set(ap)
	}
	return sliceAddr.Elem()
}

func build(objs ...interface{}) (c *cli.App, err error) {
	c = cli.NewApp()

	commands, err := buildCommands(objs...)
	if err != nil {
		return
	}

	// if it's a one-command application, there's no need for a subcommand so
	// just move the command's contents into the root object, aka the 'App'
	if len(commands) == 1 {
		c.Usage = commands[0].Usage
		c.Action = commands[0].Action
		c.Flags = commands[0].Flags
	} else {
		c.Commands = commands
		c.Flags = nil
	}
	return
}

func buildCommands(objs ...interface{}) (commands []*cli.Command, err error) {
	for _, obj := range objs {
		var command *cli.Command
		command, err = commandFromObject(obj)
		if err != nil {
			return
		}
		commands = append(commands, command)
	}
	return
}

type commandMetadata struct {
	Skipped bool
	Name    string
	Usage   string
	Hidden  bool
	Default string
}

func commandFromObject(obj interface{}) (command *cli.Command, err error) {
	if obj == nil {
		return nil, errors.New("obj is null")
	}

	// recursively dereference
	objValue := reflect.ValueOf(obj)
	objIsPointer := false
	for objValue.Kind() == reflect.Ptr {
		objValue = objValue.Elem()
		objIsPointer = true
	}

	// anonymous structs (struct{ ... }{}) are not allowed
	objType := objValue.Type()
	if objType.Name() == "" {
		return nil, errors.New("need a named struct type to determine command name")
	}

	// the first field must be an embedded cli.Command struct
	command, err = getCommand(objType.Field(0), objValue.Field(0))
	if err != nil {
		return nil, err
	}
	command.Name = strings.ToLower(objType.Name())

	for i := 1; i < objType.NumField(); i++ {
		fieldType := objType.Field(i)

		cmdmeta, err := parseMeta(fieldType.Tag.Get("cli"))
		if err != nil {
			return nil, err
		}
		if cmdmeta.Skipped {
			continue
		}

		// automatically turn fields that begin with Flag into cli.Flag objects
		if strings.HasPrefix(fieldType.Name, "Flag") {
			flag, err := flagFromType(fieldType, cmdmeta)
			if err != nil {
				return nil, errors.Wrap(err, "failed to generate flag from struct field")
			}
			command.Flags = append(command.Flags, flag)
		}
	}
	if objValue.CanAddr() {
		act, ok := objValue.Addr().Interface().(Actionable)
		if !ok {
			return command, nil
		}
		if !objIsPointer {
			return nil, errors.New("an Actionable struct is passed by value, pass by reference")
		}
		if command.Action != nil {
			return nil, errors.New("embedded cli.Command has action")
		}
		command.Action = func(ctx *cli.Context) error {
			flags := flagsForActionable(act, ctx)
			return flags.Action(ctx)
		}
	}

	return command, nil
}

func getCommand(fieldType reflect.StructField, fieldValue reflect.Value) (c *cli.Command, err error) {
	if fieldType.Name != "Command" {
		return nil, errors.New("first field must be an embedded cli.Command")
	}

	if fieldValue.Kind() != reflect.Struct {
		return nil, errors.New("expected Command field to be a struct (specifically, an embedded cli.Command struct)")
	}

	cmd, ok := fieldValue.Interface().(cli.Command)
	if !ok {
		return nil, errors.New("failed to cast Command field to a cli.Command object")
	}

	cmdmeta, err := parseMeta(fieldType.Tag.Get("cli"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to read cmdmeta tag on the embedded cli.Command struct")
	}
	cmd.Usage = cmdmeta.Usage
	cmd.Flags = []cli.Flag{}

	return &cmd, nil
}

func parseMeta(s string) (cmdmeta commandMetadata, err error) {
	// this code allows strings to be placed inside single-quotes in order to
	// escape comma characters.
	quotes := false
	sections := strings.FieldsFunc(s, func(r rune) bool {
		if r == '\'' && !quotes {
			quotes = true
		} else if r == '\'' && quotes {
			quotes = false
		}
		if r == ',' && !quotes {
			return true
		}
		return false
	})
	cmdmeta.Skipped = false
	for _, section := range sections {
		if section == "-" {
			cmdmeta.Skipped = true
			return cmdmeta, err
		}
		keyvalue := strings.SplitN(section, ":", 2)
		if len(keyvalue) == 2 {
			switch keyvalue[0] {
			case "name":
				cmdmeta.Name = keyvalue[1]
			case "usage":
				cmdmeta.Usage = strings.Trim(keyvalue[1], "'") // trim single-quotes
			case "hidden":
				cmdmeta.Hidden, err = strconv.ParseBool(keyvalue[1])
				if err != nil {
					err = errors.Wrap(err, "failed to parse 'hidden' as a bool")
				}
			case "default":
				cmdmeta.Default = keyvalue[1]
			default:
				err = errors.Errorf("unknown command tag: '%s:%s'", keyvalue[0], keyvalue[1])
			}
		} else {
			err = errors.Errorf("malformed tag: '%s'", section)
		}
		if err != nil {
			return
		}
	}
	return cmdmeta, err
}

//nolint:errcheck
func flagFromType(fieldType reflect.StructField, cmdmeta commandMetadata) (flag cli.Flag, err error) {
	var (
		name string
		env  string
	)

	if cmdmeta.Name != "" {
		name = strcase.ToKebab(cmdmeta.Name)
	} else {
		name = strcase.ToKebab(strings.TrimPrefix(fieldType.Name, "Flag"))
	}
	env = strcase.ToScreamingSnake(name)

	cmdmeta.Default = strings.Trim(cmdmeta.Default, "'")

	switch fieldType.Type.String() {
	case "int":
		def, _ := strconv.ParseInt(cmdmeta.Default, 10, 64)
		flag = &cli.IntFlag{
			Name:    name,
			EnvVars: []string{env},
			Value:   int(def),
			Hidden:  cmdmeta.Hidden,
			Usage:   cmdmeta.Usage,
		}

	case "int64":
		def, _ := strconv.ParseInt(cmdmeta.Default, 10, 64)
		flag = &cli.Int64Flag{
			Name:    name,
			EnvVars: []string{env},
			Value:   def,
			Hidden:  cmdmeta.Hidden,
			Usage:   cmdmeta.Usage,
		}

	case "uint":
		def, _ := strconv.ParseUint(cmdmeta.Default, 10, 64)
		flag = &cli.UintFlag{
			Name:    name,
			EnvVars: []string{env},
			Value:   uint(def),
			Hidden:  cmdmeta.Hidden,
			Usage:   cmdmeta.Usage,
		}

	case "uint64":
		def, _ := strconv.ParseUint(cmdmeta.Default, 10, 64)
		flag = &cli.Uint64Flag{
			Name:    name,
			EnvVars: []string{env},
			Value:   def,
			Hidden:  cmdmeta.Hidden,
			Usage:   cmdmeta.Usage,
		}

	case "float32":
		def, _ := strconv.ParseFloat(cmdmeta.Default, 32)
		flag = &cli.Float64Flag{
			Name:    name,
			EnvVars: []string{env},
			Value:   def,
			Hidden:  cmdmeta.Hidden,
			Usage:   cmdmeta.Usage,
		}

	case "float64":
		def, _ := strconv.ParseFloat(cmdmeta.Default, 64)
		flag = &cli.Float64Flag{
			Name:    name,
			EnvVars: []string{env},
			Value:   def,
			Hidden:  cmdmeta.Hidden,
			Usage:   cmdmeta.Usage,
		}

	case "bool":
		def, _ := strconv.ParseBool(cmdmeta.Default)
		if !def {
			flag = &cli.BoolFlag{
				Name:    name,
				EnvVars: []string{env},
				Hidden:  cmdmeta.Hidden,
				Usage:   cmdmeta.Usage,
			}
		} else {
			flag = &cli.BoolFlag{
				Name:    name,
				EnvVars: []string{env},
				Value:   true,
				Hidden:  cmdmeta.Hidden,
				Usage:   cmdmeta.Usage,
			}
		}

	case "string":
		flag = &cli.StringFlag{
			Name:    name,
			EnvVars: []string{env},
			Value:   cmdmeta.Default,
			Hidden:  cmdmeta.Hidden,
			Usage:   cmdmeta.Usage,
		}

	case "time.Duration":
		def, _ := time.ParseDuration(cmdmeta.Default)
		flag = &cli.DurationFlag{
			Name:    name,
			EnvVars: []string{env},
			Value:   def,
			Hidden:  cmdmeta.Hidden,
			Usage:   cmdmeta.Usage,
		}

	case "[]int":
		var def *cli.IntSlice // must remain nil if unset
		if cmdmeta.Default != "" {
			var defs []int
			for _, s := range strings.Split(cmdmeta.Default, ",") {
				d, _ := strconv.Atoi(s)
				defs = append(defs, d)
			}
			def = cli.NewIntSlice(defs...)
		}
		flag = &cli.IntSliceFlag{
			Name:    name,
			EnvVars: []string{env},
			Value:   def,
			Hidden:  cmdmeta.Hidden,
			Usage:   cmdmeta.Usage,
		}

	case "[]int64":
		var def *cli.Int64Slice // must remain nil if unset
		if cmdmeta.Default != "" {
			var defs []int64
			for _, s := range strings.Split(cmdmeta.Default, ",") {
				d, _ := strconv.Atoi(s)
				defs = append(defs, int64(d))
			}
			def = cli.NewInt64Slice(defs...)
		}
		flag = &cli.Int64SliceFlag{
			Name:    name,
			EnvVars: []string{env},
			Value:   def,
			Hidden:  cmdmeta.Hidden,
			Usage:   cmdmeta.Usage,
		}

	// urfave/cli does not have unsigned types yet
	// case "[]uint":
	// 	flag = &cli.IntSliceFlag{
	// 		Name:   name,
	// 		EnvVars: []string{env},
	// 		Hidden: cmdmeta.Hidden,
	// 		Usage:  cmdmeta.Usage,
	// 	}

	// case "[]uint64":
	// 	flag = &cli.Int64SliceFlag{
	// 		Name:   name,
	// 		EnvVars: []string{env},
	// 		Hidden: cmdmeta.Hidden,
	// 		Usage:  cmdmeta.Usage,
	// 	}

	case "[]string":
		var def *cli.StringSlice // must remain nil if unset
		if cmdmeta.Default != "" {
			def = cli.NewStringSlice(strings.Split(cmdmeta.Default, ",")...)
		}
		flag = &cli.StringSliceFlag{
			Name:    name,
			EnvVars: []string{env},
			Value:   def,
			Hidden:  cmdmeta.Hidden,
			Usage:   cmdmeta.Usage,
		}

	default:
		if reflect.PointerTo(fieldType.Type).Implements(TextUnmarshalerType) {
			flag = &cli.StringFlag{
				Name:    name,
				EnvVars: []string{env},
				Value:   cmdmeta.Default,
				Hidden:  cmdmeta.Hidden,
				Usage:   cmdmeta.Usage,
			}
		} else {
			err = errors.Errorf("unsupported flag generator type: %s", fieldType.Type.String())
		}
	}

	return flag, err
}

var TextUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
