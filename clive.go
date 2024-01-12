package clive

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

type (
	Actionable interface {
		Action(*cli.Context) error
	}
	WithVersion interface {
		Version() string
	}
	WithDescription interface {
		Description() string
	}
)

type CommandLike interface {
	Root(ctx *cli.Context) interface{}
	Parent(ctx *cli.Context) interface{}
}

type RunFunc func(*Command, *cli.Context) error

type Command struct {
	*cli.Command
	run         RunFunc
	parentPath  string
	currentPath string
}

func (c *Command) Root(ctx *cli.Context) interface{} {
	app, ok := ctx.App.Metadata["cliveRoot"]
	if !ok {
		panic("no cliveRoot metadata in App")
	}
	return app
}

func (c *Command) Parent(ctx *cli.Context) interface{} {
	parent, ok := ctx.App.Metadata[c.parentPath]
	if !ok {
		panic(fmt.Sprintf("no parent (%s) metadata in App", c.parentPath))
	}
	return parent
}

func (c *Command) Current(ctx *cli.Context) interface{} {
	current, ok := ctx.App.Metadata[c.currentPath]
	if !ok {
		panic(fmt.Sprintf("no current (%s) metadata in App", c.currentPath))
	}
	return current
}

func ErrCommandNotImplemented() error {
	return errors.New("command not implemented")
}

func (c *Command) Action(ctx *cli.Context) error {
	if c.run != nil {
		return c.run(c, ctx)
	}
	if ctx.Command != nil {
		cli.ShowAppHelp(ctx)
	} else {
		cli.ShowSubcommandHelp(ctx)
	}
	return ErrCommandNotImplemented()
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

func flagsForActionable(act Actionable, c *cli.Context) (Actionable, error) {

	objValue := reflect.ValueOf(act)
	for objValue.Kind() == reflect.Ptr {
		objValue = objValue.Elem()
	}

	objType := objValue.Type()

	err := flagsForValue(&objValue, objType, c)

	return act, err
}

func flagsForValue(obj *reflect.Value, objType reflect.Type, c *cli.Context) error {
	args := c.Args().Slice()
	for i := 1; i < objType.NumField(); i++ {
		fieldType := objType.Field(i)
		if fieldType.Name == "Subcommands" || (fieldType.Name == "Run" && fieldType.Type == reflect.TypeOf((RunFunc)(nil))) {
			continue
		}
		cmdmeta, err := parseMeta(fieldType)
		if err != nil {
			return err
		}
		if cmdmeta.Skipped {
			continue
		}
		field := obj.FieldByName(fieldType.Name).Addr()
		var setFrom string
		if cmdmeta.Positional {
			if len(args) == 0 {
				if !cmdmeta.Required {
					err = cmdmeta.SetValueFromString(field, cmdmeta.Default)
					if err != nil {
						setFrom = fmt.Sprintf("from default value %s", cmdmeta.Default)
					}
				} else {
					err = errors.New("too few positional arguments")
				}
			} else {
				if cmdmeta.IsVariadic() {
					err = cmdmeta.SetValueFromStrings(field, args)
					args = []string{}
				} else {
					err = cmdmeta.SetValueFromString(field, args[0])
					args = args[1:]
				}
			}
			if err != nil {
				setFrom = fmt.Sprintf("positional argument %s %s", strcase.ToScreamingSnake(cmdmeta.Name), setFrom)
			}
		} else {
			err = cmdmeta.SetValueFromContext(field, cmdmeta.Name, c)
			if err != nil {
				setFrom = fmt.Sprintf("from flag %s", cmdmeta.Name)
			}
		}
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to set field %s (type %s) from %s", fieldType.Name, fieldType.Type.String(), setFrom))
		}
	}
	return nil
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
	c.Metadata = make(map[string]interface{})

	commands, err := buildCommands(c, "", objs...)
	if err != nil {
		return
	}

	// if it's a one-command application, there's no need for a subcommand so
	// just move the command's contents into the root object, aka the 'App'
	if len(commands) == 1 {
		c.Usage = commands[0].Usage
		c.Description = commands[0].Description
		c.Before = commands[0].Before
		c.Action = commands[0].Action
		c.Flags = commands[0].Flags
		c.Commands = commands[0].Subcommands
		c.Metadata["cliveRoot"] = objs[0]
		if versioned, ok := objs[0].(WithVersion); ok {
			c.Version = versioned.Version()
		}
	} else {
		c.Commands = commands
		c.Flags = nil
	}

	// c.Before = func(ctx *cli.Context) error {
	// 	log.Println("App before")
	// 	return nil
	// }
	return
}

func buildCommands(c *cli.App, parentCommandPath string, objs ...interface{}) (commands []*cli.Command, err error) {
	for _, obj := range objs {
		var command *cli.Command
		command, err = commandFromObject(c, parentCommandPath, obj)
		if err != nil {
			return
		}
		commands = append(commands, command)
	}
	return
}

func buildSubcommands(c *cli.App, parentCommandPath string, subcommandsField reflect.Value) (commands []*cli.Command, err error) {
	subcommandsFieldValue := subcommandsField
	for subcommandsFieldValue.Kind() == reflect.Ptr {
		subcommandsFieldValue = subcommandsFieldValue.Elem()
	}

	subcommandsType := subcommandsFieldValue.Type()
	var subcommands = make([]interface{}, 0, subcommandsType.NumField())
	for i := 0; i < subcommandsType.NumField(); i++ {
		subcommandFieldType := subcommandsType.Field(i)
		subcommand := subcommandsFieldValue.Field(i)
		if subcommandFieldType.Type.Kind() != reflect.Pointer {
			return nil, fmt.Errorf("type of subcommand (%s) for %s is passed by value, not by reference", subcommandFieldType.Type.Name(), parentCommandPath)
		}
		if subcommandFieldType.Type.Elem().Kind() != reflect.Struct {
			return nil, fmt.Errorf("type of subcommand (%v) for %s is a double pointer (Kind: %v), should be a pointer to struct", subcommandFieldType.Type.Name(), parentCommandPath, subcommand.Kind())
		}
		if subcommand.IsNil() {
			subcommand.Set(reflect.New(subcommandFieldType.Type.Elem()))
		}
		subcommands = append(subcommands, subcommand.Interface())
	}
	return buildCommands(c, parentCommandPath, subcommands...)
}

type commandMetadata struct {
	TypeInterface
	Name       string
	Envs       []string
	Usage      string
	Hidden     bool
	Default    string
	Skipped    bool
	Positional bool
	Required   bool
}

func commandFromObject(c *cli.App, parentCommandPath string, obj interface{}) (*cli.Command, error) {
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
	if !objValue.CanAddr() || !objIsPointer {
		return nil, fmt.Errorf("command struct %s is passed by value, pass by reference", objValue.Type().Name())
	}

	// anonymous structs (struct{ ... }{}) are not allowed
	objType := objValue.Type()

	// the first field must be an embedded *Command struct
	command, err := getCommand(objType.Field(0), objValue.Field(0))
	if err != nil {
		return nil, err
	}

	// name from tags takes precedence
	if command.Name == "" {
		command.Name = strings.ToLower(objType.Name())
	}

	commandPath := "/"
	if parentCommandPath != "" {
		commandPath = fmt.Sprintf("%s%s/", commandPath, parentCommandPath)
		command.parentPath = parentCommandPath
	}
	commandPath = fmt.Sprintf("%s%s", commandPath, command.Name)
	command.currentPath = commandPath

	command.Before = func(ctx *cli.Context) error {
		act := ctx.App.Metadata[commandPath].(Actionable)
		flags, err := flagsForActionable(act, ctx)
		if err == nil {
			ctx.App.Metadata[commandPath] = flags
		} else {
			cli.ShowSubcommandHelp(ctx)
		}
		return err
	}
	command.Command.Action = func(ctx *cli.Context) error {
		act := ctx.App.Metadata[commandPath].(Actionable)
		return act.Action(ctx)
	}

	act, ok := objValue.Addr().Interface().(Actionable)
	if !ok {
		return nil, fmt.Errorf("command struct %s must implement Actionable", objValue.Type().Name())
	}
	c.Metadata[commandPath] = act

	if desc, ok := objValue.Addr().Interface().(WithDescription); ok {
		command.Description = desc.Description()
	}

	var positionals []commandMetadata

	for i := 1; i < objType.NumField(); i++ {
		fieldType := objType.Field(i)
		if fieldType.Name == "Subcommands" {
			command.Subcommands, err = buildSubcommands(c, commandPath, objValue.Field(i).Addr())
			if err != nil {
				return nil, err
			}
			continue
		}
		if fieldType.Name == "Run" && fieldType.Type == reflect.TypeOf((RunFunc)(nil)) {
			command.run = objValue.Field(i).Interface().(RunFunc)
			continue
		}

		var cmdmeta commandMetadata
		cmdmeta, err = parseMeta(fieldType)
		if err != nil {
			return nil, err
		}
		if cmdmeta.Skipped {
			continue
		}

		if cmdmeta.Positional {
			positionals = append(positionals, cmdmeta)
		} else {
			// automatically turn fields that begin with Flag into cli.Flag objects
			var flag cli.Flag
			flag, err = cmdmeta.NewFlag(cmdmeta)
			if err != nil {
				return nil, err
			}
			command.Flags = append(command.Flags, flag)
		}
	}
	command.Args = len(positionals) != 0
	optionalStarted := false
	variadicStarted := false
	var positionalUsage []string
	for _, positional := range positionals {
		if variadicStarted {
			return nil, fmt.Errorf("cant add positional argument %s after a variadic (slice of x) argument", positional.Name)
		}
		if positional.Hidden {
			return nil, fmt.Errorf("positional argument %s cannot be Hidden", positional.Name)
		}
		usage := strcase.ToScreamingSnake(positional.Name)
		if positional.IsVariadic() {
			variadicStarted = true
			usage = fmt.Sprintf("%s [%[1]s]", usage)
		}
		optional := !positional.Required
		if optional {
			optionalStarted = true
			usage = fmt.Sprintf("[%s]", usage)
		} else {
			if optionalStarted {
				return nil, fmt.Errorf("positional argument %s cannot be non-optional after an optional argument", positional.Name)
			}
		}
		positionalUsage = append(positionalUsage, usage)
	}
	command.ArgsUsage = strings.Join(positionalUsage, " ")
	return command.Command, nil
}

func getCommand(fieldType reflect.StructField, fieldValue reflect.Value) (c *Command, err error) {
	if fieldType.Name != "Command" {
		return nil, fmt.Errorf("first field must be an embedded cli.Command, got %s", fieldType.Name)
	}

	if fieldValue.Kind() != reflect.Pointer {
		return nil, errors.New("expected Command field to be a pointer (specifically, an embedded *clive.Command struct pointer)")
	}

	if fieldValue.IsNil() {
		fieldValue.Set(reflect.ValueOf(&Command{}))
	}

	cmd, ok := fieldValue.Interface().(*Command)
	if !ok {
		return nil, errors.New("failed to cast Command field to a clive.Command object")
	}
	if cmd.Command == nil {
		cmd.Command = &cli.Command{}
	}

	cmdmeta, err := parseMeta(fieldType)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read cmdmeta tag on the embedded clive.Command struct pointer")
	}
	if cmdmeta.Name != "" {
		cmd.Name = cmdmeta.Name
	}
	cmd.Usage = cmdmeta.Usage
	cmd.Flags = []cli.Flag{}

	return cmd, nil
}

func parseMeta(fieldType reflect.StructField) (cmdmeta commandMetadata, err error) {
	s := fieldType.Tag.Get("cli")

	cmdmeta.Skipped = false
	if s == "-" {
		cmdmeta.Skipped = true
		return cmdmeta, err
	}
	cmdmeta.Required = false
	cmdmeta.Positional = false
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
	requiredSetFromTags := false
	for _, section := range sections {
		if section == "positional" {
			cmdmeta.Positional = true
			continue
		}
		keyvalue := strings.SplitN(section, ":", 2)
		if len(keyvalue) == 2 {
			keyvalue[1] = strings.Trim(keyvalue[1], "'")
			switch keyvalue[0] {
			case "name":
				cmdmeta.Name = keyvalue[1]
			case "usage":
				cmdmeta.Usage = keyvalue[1] // trim single-quotes
			case "required":
				cmdmeta.Required, err = strconv.ParseBool(keyvalue[1])
				if err != nil {
					err = errors.Wrap(err, "failed to parse 'required' as a bool")
				}
				requiredSetFromTags = true
			case "env":
				cmdmeta.Envs = strings.Split(keyvalue[1], ",")
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
	if cmdmeta.Positional {
		if !requiredSetFromTags {
			cmdmeta.Required = cmdmeta.Default == ""
		}
	}
	if fieldType.Type != reflect.TypeOf((*Command)(nil)) {
		cmdmeta.TypeInterface, err = flagType(fieldType)
		if err != nil {
			err = errors.Wrapf(err, "cant find type for %s field", cmdmeta.Name)
			return
		}
		if cmdmeta.Name == "" {
			cmdmeta.Name = fieldType.Name
		}
	}
	if cmdmeta.Name != "" {
		cmdmeta.Name = strcase.ToKebab(cmdmeta.Name)
	}
	if len(cmdmeta.Envs) == 0 {
		cmdmeta.Envs = []string{
			strcase.ToScreamingSnake(cmdmeta.Name),
		}
	}
	return cmdmeta, err
}
