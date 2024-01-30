package clive

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/urfave/cli/v2"
)

type (
	HasBefore interface {
		Before(*cli.Context) error
	}
	Actionable interface {
		Action(*cli.Context) error
	}
	HasAfter interface {
		After(*cli.Context) error
	}
	WithVersion interface {
		Version() string
	}
	WithDescription interface {
		Description() string
	}
	HasVariants interface {
		Variants() []string
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

type MethodNotFoundError struct {
	methodName string
}

func (e *MethodNotFoundError) Error() string {
	return fmt.Sprintf("flag required, but no suitable fallback method found (%s)", e.methodName)
}

var ErrNil = errors.New("obj is n ull")

type ByValueError struct {
	Type string
}

func (e *ByValueError) Error() string {
	return fmt.Sprintf("command struct %s is passed by value, pass by reference", e.Type)
}

type ActionableNotImplementedError struct {
	Type string
}

func (e *ActionableNotImplementedError) Error() string {
	return fmt.Sprintf("command struct %s must implement Actionable", e.Type)
}

type WrongFirstFieldError struct {
	NumFields int

	FieldName string
	Type      string
}

func (e *WrongFirstFieldError) Error() string {
	return fmt.Sprintf(`
	command struct:
	* should have at least one field (have %d)
	* its first field must be an embedded *clive.Command (name: %s, type: %s)
	`, e.NumFields, e.FieldName, e.Type)
}

type HiddenPositionalError struct {
	Name string
}

func (e *HiddenPositionalError) Error() string {
	return fmt.Sprintf("positional argument %s cannot be Hidden", e.Name)
}

type PositionalAfterVariadicError struct {
	CurrentName string
	FirstName   string
}

func (e *PositionalAfterVariadicError) Error() string {
	return fmt.Sprintf("cant add positional argument %s after variadic (slice of x) argument %s", e.CurrentName, e.FirstName)
}

// Build constructs a urfave/cli App from an instance of a decorated struct
// Since it is designed to be used 1. on initialization and; 2. with static data
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

func flagFromUnsetMethod(obj *reflect.Value, fieldType reflect.StructField, c *cli.Context) (err error) {
	methodName := "On" + fieldType.Name + "Unset"
	unset := obj.Addr().MethodByName(methodName)
	if unset.IsValid() && unset.CanInterface() {
		if unsetFunc, set := unset.Interface().(func() error); set {
			err = unsetFunc()
		} else if unsetFunc, set := unset.Interface().(func(*cli.Context) error); set {
			err = unsetFunc(c)
		} else {
			err = &MethodNotFoundError{methodName: methodName}
		}
	}
	return
}

func flagsForValue(obj *reflect.Value, objType reflect.Type, c *cli.Context) error {
	args := c.Args().Slice()
	hadPositionals := false
	for i := 1; i < objType.NumField(); i++ {
		fieldType := objType.Field(i)
		if fieldType.Name == "Subcommands" || (fieldType.Name == "Run" && fieldType.Type == reflect.TypeOf((RunFunc)(nil))) {
			continue
		}
		cmdMeta, err := parseMeta(fieldType)
		if err != nil {
			return err
		}
		if cmdMeta.Skipped {
			continue
		}
		field := obj.FieldByName(fieldType.Name).Addr()
		var setFrom string
		if cmdMeta.Positional {
			hadPositionals = true
			if len(args) == 0 {
				if !cmdMeta.Required {
					if cmdMeta.Default != nil {
						err = cmdMeta.SetValueFromString(field, *cmdMeta.Default)
						if err != nil {
							setFrom = fmt.Sprintf("from default value %s", *cmdMeta.Default)
						}
					}
					err = flagFromUnsetMethod(obj, fieldType, c)
					if err != nil {
						setFrom = "from fallback method"
					}
				} else {
					err = errors.New("too few positional arguments")
				}
			} else {
				if cmdMeta.IsVariadic() {
					err = cmdMeta.SetValueFromStrings(field, args)
					args = []string{}
				} else {
					err = cmdMeta.SetValueFromString(field, args[0])
					args = args[1:]
				}
			}
			if err != nil {
				setFrom = fmt.Sprintf("positional argument %s %s", strcase.ToScreamingSnake(cmdMeta.Name), setFrom)
			}
		} else {
			if c.IsSet(cmdMeta.Name) || cmdMeta.Default != nil {
				err = cmdMeta.SetValueFromContext(field, cmdMeta.Name, c)
			} else {
				err = flagFromUnsetMethod(obj, fieldType, c)
			}
			if err != nil {
				setFrom = fmt.Sprintf("from flag %s", cmdMeta.Name)
			}
		}
		if err != nil {
			return fmt.Errorf("failed to set field %s (type %s) from %s: %s", fieldType.Name, fieldType.Type.String(), setFrom, err.Error())
		}
	}
	if hadPositionals && len(args) > 0 {
		return fmt.Errorf("too many arguments: %d left unparsed: %s", len(args), strings.Join(args, " "))
	}
	return nil
}

func build(objs ...interface{}) (c *cli.App, err error) {
	c = cli.NewApp()
	c.Metadata = make(map[string]interface{})
	c.HideHelpCommand = true

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
	Default    *string
	Skipped    bool
	Positional bool
	Required   bool
}

func commandFromObject(c *cli.App, parentCommandPath string, obj interface{}) (*cli.Command, error) {
	if obj == nil {
		return nil, ErrNil
	}

	// recursively dereference
	objValue := reflect.ValueOf(obj)
	objIsPointer := false
	for objValue.Kind() == reflect.Ptr {
		objValue = objValue.Elem()
		objIsPointer = true
	}
	if !objValue.CanAddr() || !objIsPointer {
		return nil, &ByValueError{objValue.Type().Name()}
	}

	objType := objValue.Type()

	if objType.NumField() == 0 {
		return nil, &WrongFirstFieldError{NumFields: 0}
	}

	// the first field must be an embedded *Command struct
	command, err := getCommand(objType.Field(0), objValue.Field(0))
	if err != nil {
		if wffe, ok := err.(*WrongFirstFieldError); ok {
			wffe.NumFields = objType.NumField()
		}
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
		obj := ctx.App.Metadata[commandPath]
		act := obj.(Actionable)
		flags, err := flagsForActionable(act, ctx)
		if err == nil {
			ctx.App.Metadata[commandPath] = flags
		} else {
			cli.ShowSubcommandHelp(ctx)
			return err
		}
		if before, ok := obj.(HasBefore); ok {
			err = before.Before(ctx)
		}
		return err
	}
	command.Command.Action = func(ctx *cli.Context) error {
		act := ctx.App.Metadata[commandPath].(Actionable)
		return act.Action(ctx)
	}
	command.Command.After = func(ctx *cli.Context) (err error) {
		obj := ctx.App.Metadata[commandPath]
		if after, ok := obj.(HasAfter); ok {
			err = after.After(ctx)
		}
		return
	}

	act, ok := objValue.Addr().Interface().(Actionable)
	if !ok {
		return nil, &ActionableNotImplementedError{objValue.Type().Name()}
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

		var cmdMeta commandMetadata
		cmdMeta, err = parseMeta(fieldType)
		if err != nil {
			return nil, err
		}
		if cmdMeta.Skipped {
			continue
		}

		if cmdMeta.Positional {
			positionals = append(positionals, cmdMeta)
		} else {
			// automatically turn fields that begin with Flag into cli.Flag objects
			var flag cli.Flag
			flag, err = cmdMeta.NewFlag(cmdMeta)
			if err != nil {
				return nil, err
			}

			command.Flags = append(command.Flags, flag)
		}
	}
	command.Args = len(positionals) != 0
	optionalStarted := false
	var variadicStarted *string = nil
	var positionalUsage []string
	for _, positional := range positionals {
		if variadicStarted != nil {
			return nil, &PositionalAfterVariadicError{CurrentName: positional.Name, FirstName: *variadicStarted}
		}
		if positional.Hidden {
			return nil, &HiddenPositionalError{positional.Name}
		}
		usage := strcase.ToScreamingSnake(positional.Name)
		if positional.IsVariadic() {
			variadicStarted = new(string)
			*variadicStarted = positional.Name
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
	command.HideHelpCommand = true
	return command.Command, nil
}

func getCommand(fieldType reflect.StructField, fieldValue reflect.Value) (c *Command, err error) {
	if fieldType.Name != "Command" || fieldType.Type != reflect.TypeOf((*Command)(nil)) {
		return nil, &WrongFirstFieldError{
			NumFields: 0,
			FieldName: fieldType.Name,
			Type:      fieldType.Type.String(),
		}
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

	cmdMeta, err := parseMeta(fieldType)
	if err != nil {
		return nil, fmt.Errorf("failed to read cmdMeta tag on the embedded clive.Command struct pointer: %s", err.Error())
	}
	if cmdMeta.Name != "" {
		cmd.Name = cmdMeta.Name
	}
	cmd.Usage = cmdMeta.Usage
	cmd.Flags = []cli.Flag{}

	return cmd, nil
}

func parseMeta(fieldType reflect.StructField) (cmdMeta commandMetadata, err error) {
	s := fieldType.Tag.Get("cli")

	cmdMeta.Skipped = false
	if s == "-" {
		cmdMeta.Skipped = true
		return cmdMeta, err
	}
	cmdMeta.Required = false
	cmdMeta.Positional = false
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
			cmdMeta.Positional = true
			continue
		}
		keyValue := strings.SplitN(section, ":", 2)
		if len(keyValue) == 2 {
			keyValue[1] = strings.Trim(keyValue[1], "'")
			switch keyValue[0] {
			case "name":
				cmdMeta.Name = keyValue[1]
			case "usage":
				cmdMeta.Usage = keyValue[1] // trim single-quotes
			case "required":
				cmdMeta.Required, err = strconv.ParseBool(keyValue[1])
				if err != nil {
					err = fmt.Errorf("failed to parse 'required' as a bool %s", err.Error())
				}
				requiredSetFromTags = true
			case "env":
				cmdMeta.Envs = strings.Split(keyValue[1], ",")
			case "hidden":
				cmdMeta.Hidden, err = strconv.ParseBool(keyValue[1])
				if err != nil {
					err = fmt.Errorf("failed to parse 'hidden' as a bool %s", err.Error())
				}
			case "default":
				cmdMeta.Default = new(string)
				*cmdMeta.Default = keyValue[1]
			case "entrypoint":

			default:
				err = fmt.Errorf("unknown command tag: '%s:%s'", keyValue[0], keyValue[1])
			}
		} else {
			err = fmt.Errorf("malformed tag: '%s'", section)
		}
		if err != nil {
			return
		}
	}
	if cmdMeta.Positional {
		if !requiredSetFromTags {
			cmdMeta.Required = cmdMeta.Default == nil
		}
	}
	if fieldType.Type != reflect.TypeOf((*Command)(nil)) {
		cmdMeta.TypeInterface, err = flagType(fieldType)
		if err != nil {
			err = fmt.Errorf("cant find type for %s field: %s", cmdMeta.Name, err.Error())
			return
		}
		if cmdMeta.Name == "" {
			cmdMeta.Name = fieldType.Name
		}
		ft := fieldType.Type
		for ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		ft = reflect.PointerTo(ft)
		if ft.Implements(Reflected[HasVariants]()) {
			vars := reflect.Zero(ft).Interface().(HasVariants).Variants()
			var usageArr []string
			if len(cmdMeta.Usage) > 0 {
				usageArr = append(usageArr, cmdMeta.Usage)
			}
			usageArr = append(usageArr, fmt.Sprintf("possible values: [%s]", strings.Join(vars, ", ")))
			cmdMeta.Usage = strings.Join(usageArr, ", ")
		}
	}
	if cmdMeta.Name != "" {
		cmdMeta.Name = strcase.ToKebab(cmdMeta.Name)
	}
	if len(cmdMeta.Envs) == 0 {
		cmdMeta.Envs = []string{
			strcase.ToScreamingSnake(cmdMeta.Name),
		}
	}
	return cmdMeta, err
}
