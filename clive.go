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
	HasSubcommand interface {
		Subcommand(c *cli.App, parentCommandPath string) *cli.Command
	}
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

	root := c.Root(ctx)
	current := c.Current(ctx)

	var err error
	if root == current {
		err = cli.ShowAppHelp(ctx)
	} else {
		parent := c.Parent(ctx)
		if parent == root {
			err = cli.ShowSubcommandHelp(ctx)
		} else {
			err = cli.ShowSubcommandHelp(ctx)
		}
	}
	if err == nil {
		err = ErrCommandNotImplemented()
	}
	return err
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

type BuildOptions struct {
	EnvPrefix string
}

var DefaultBuildOptions = BuildOptions{
	EnvPrefix: "",
}

// Build constructs a urfave/cli App from an instance of a decorated struct
// Since it is designed to be used 1. on initialization and; 2. with static data
// that is compile-time only - it does not return an error but instead panics.
// The idea is you will do all your setup once and as long as it doesn't change
// this will never break, so there is little need to pass errors back.
func Build(obj interface{}) (c *cli.App) {
	return BuildCustom(obj, DefaultBuildOptions)
}

func BuildCustom(obj interface{}, o BuildOptions) (c *cli.App) {
	c, err := build(obj, &o)
	if err != nil {
		panic(err)
	}
	return
}

func flagsForActionable(act Actionable, c *cli.Context, bo *BuildOptions) (Actionable, error) {

	objValue := reflect.ValueOf(act)
	for objValue.Kind() == reflect.Ptr {
		objValue = objValue.Elem()
	}

	objType := objValue.Type()

	err := flagsForValue(&objValue, objType, c, bo)

	return act, err
}

func flagsForValue(obj *reflect.Value, objType reflect.Type, c *cli.Context, bo *BuildOptions) error {
	args := c.Args().Slice()
	hadPositionals := false
	for i := 1; i < objType.NumField(); i++ {
		fieldType := objType.Field(i)
		if fieldType.Name == "Subcommands" || (fieldType.Name == "Run" && fieldType.Type == reflect.TypeOf((RunFunc)(nil))) {
			continue
		}
		var flieldMetadata []commandMetadata
		err := parseFieldOrPositional("", []int{i}, fieldType, &flieldMetadata, &flieldMetadata, bo)
		if err != nil {
			return err
		}
		for _, cmdMeta := range flieldMetadata {
			if cmdMeta.Skipped {
				continue
			}
			currentObj := obj.Addr()
			var currentField reflect.Value
			for accessIndex, fieldIndex := range cmdMeta.Accesses {
				if accessIndex > 0 {
					currentObj = currentField
				}
				currentField = currentObj.Elem().Field(fieldIndex).Addr()
			}
			var setFrom string
			if cmdMeta.Positional {
				hadPositionals = true
				if len(args) == 0 {
					if !cmdMeta.Required {
						if cmdMeta.Default != nil {
							err = cmdMeta.SetValueFromString(currentField, *cmdMeta.Default)
							if err != nil {
								setFrom = fmt.Sprintf("from default value %s", *cmdMeta.Default)
							}
						}
					} else {
						err = errors.New("too few positional arguments")
					}
				} else {
					if cmdMeta.IsVariadic() {
						err = cmdMeta.SetValueFromStrings(currentField, args)
						args = []string{}
					} else {
						err = cmdMeta.SetValueFromString(currentField, args[0])
						args = args[1:]
					}
				}
				if err != nil {
					setFrom = fmt.Sprintf("positional argument %s %s", strcase.ToScreamingSnake(cmdMeta.Name), setFrom)
				}
			} else {
				if c.IsSet(cmdMeta.Name) || cmdMeta.Default != nil {
					err = cmdMeta.SetValueFromContext(currentField, cmdMeta.Name, c)
				}
				if err != nil {
					setFrom = fmt.Sprintf("from flag %s", cmdMeta.Name)
				}
			}
			if err != nil {
				return fmt.Errorf("failed to set field %s (type %s) from %s: %s", fieldType.Name, fieldType.Type.String(), setFrom, err.Error())
			}
		}
	}
	if hadPositionals && len(args) > 0 {
		return fmt.Errorf("too many arguments: %d left unparsed: %s", len(args), strings.Join(args, " "))
	}
	return nil
}

func build(obj interface{}, bo *BuildOptions) (c *cli.App, err error) {
	c = cli.NewApp()
	c.Metadata = make(map[string]interface{})
	c.HideHelpCommand = true

	commands, err := buildCommands(c, "", bo, obj)
	if err != nil {
		return
	}

	// if it's a one-command application, there's no need for a subcommand so
	// just move the command's contents into the root object, aka the 'App'
	if len(commands) != 1 {
		panic("this should never happen")
	}
	command := commands[0]
	c.Usage = command.Usage
	c.Description = command.Description
	c.Before = command.Before
	c.Action = command.Action
	c.Flags = command.Flags
	c.Commands = command.Subcommands
	c.Metadata["cliveRoot"] = obj
	if versioned, ok := obj.(WithVersion); ok {
		c.Version = versioned.Version()
	}
	c.UseShortOptionHandling = command.UseShortOptionHandling
	return
}

func buildCommands(c *cli.App, parentCommandPath string, bo *BuildOptions, objs ...interface{}) (commands []*cli.Command, err error) {
	for _, obj := range objs {
		var command *cli.Command
		command, err = commandFromObject(c, parentCommandPath, obj, bo)
		if err != nil {
			return
		}
		commands = append(commands, command)
	}
	return
}

func buildSubcommands(c *cli.App, parentCommandPath string, subcommandsField reflect.Value, bo *BuildOptions) (commands []*cli.Command, err error) {
	subcommandsFieldValue := subcommandsField
	for subcommandsFieldValue.Kind() == reflect.Ptr {
		subcommandsFieldValue = subcommandsFieldValue.Elem()
	}

	subcommandsType := subcommandsFieldValue.Type()
	var subcommands = make([]interface{}, 0, subcommandsType.NumField())
	var cmds []*cli.Command
	for i := 0; i < subcommandsType.NumField(); i++ {
		subcommandFieldType := subcommandsType.Field(i)
		subcommand := subcommandsFieldValue.Field(i)
		if subcommandFieldType.Type.Kind() == reflect.Struct {
			cmds, err = buildSubcommands(c, parentCommandPath, subcommand, bo)
			if err != nil {
				return
			}
			commands = append(commands, cmds...)
			continue
		}

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
	cmds, err = buildCommands(c, parentCommandPath, bo, subcommands...)
	if err != nil {
		return
	}
	commands = append(commands, cmds...)
	return
}

type commandMetadata struct {
	TypeInterface
	Name       string
	Envs       []string
	Aliases    []string
	Usage      string
	Hidden     bool
	Default    *string
	Skipped    bool
	Positional bool
	Inline     bool
	Required   bool
	Accesses   []int

	UseShortOptions bool
}

func commandFromObject(c *cli.App, parentCommandPath string, obj interface{}, bo *BuildOptions) (*cli.Command, error) {
	if sc, ok := obj.(HasSubcommand); ok {
		return sc.Subcommand(c, parentCommandPath), nil
	}
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
	command, err := getCommand(objType.Field(0), objValue.Field(0), bo)
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
		flags, err := flagsForActionable(act, ctx, bo)
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
	var flags []commandMetadata

	for i := 1; i < objType.NumField(); i++ {
		fieldType := objType.Field(i)
		if fieldType.Name == "Subcommands" {
			command.Subcommands, err = buildSubcommands(c, commandPath, objValue.Field(i).Addr(), bo)
			if err != nil {
				return nil, err
			}
			continue
		}
		if fieldType.Name == "Run" && fieldType.Type == reflect.TypeOf((RunFunc)(nil)) {
			command.run = objValue.Field(i).Interface().(RunFunc)
			continue
		}
		parseFieldOrPositional("", nil, fieldType, &positionals, &flags, bo)
	}
	for _, flagMeta := range flags {
		var flag cli.Flag
		flag, err = flagMeta.NewFlag(flagMeta)
		if err != nil {
			return nil, err
		}
		command.Flags = append(command.Flags, flag)
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

func getCommand(fieldType reflect.StructField, fieldValue reflect.Value, bo *BuildOptions) (c *Command, err error) {
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

	cmdMeta, err := parseMeta("", nil, fieldType, bo)
	if err != nil {
		return nil, fmt.Errorf("failed to read cmdMeta tag on the embedded clive.Command struct pointer: %s", err.Error())
	}
	if cmdMeta.Name != "" {
		cmd.Name = cmdMeta.Name
	}
	cmd.Usage = cmdMeta.Usage
	cmd.Aliases = cmdMeta.Aliases
	cmd.Flags = []cli.Flag{}
	cmd.UseShortOptionHandling = cmdMeta.UseShortOptions

	return cmd, nil
}

func parseFieldOrPositional(prefix string, accesses []int, fieldType reflect.StructField, positionals *[]commandMetadata, flags *[]commandMetadata, bo *BuildOptions) (err error) {
	var cmdMeta commandMetadata
	cmdMeta, err = parseMeta(prefix, accesses, fieldType, bo)
	if err != nil {
		return
	}
	if cmdMeta.Skipped {
		return
	}
	if cmdMeta.Inline {
		structType := fieldType.Type
		if structType.Kind() != reflect.Struct {
			err = fmt.Errorf("inline field %s is not a struct", fieldType.Name)
			return
		}
		for i := 0; i < structType.NumField(); i++ {
			fT := structType.Field(i)

			fAccesses := make([]int, len(cmdMeta.Accesses)+1)
			copy(fAccesses, cmdMeta.Accesses)
			fAccesses[len(fAccesses)-1] = i

			err = parseFieldOrPositional(cmdMeta.Name, fAccesses, fT, positionals, flags, bo)
			if err != nil {
				err = fmt.Errorf("parsing inline field %s: %w", fieldType.Name, err)
				return
			}
		}
		return
	}

	if cmdMeta.Positional {
		*positionals = append(*positionals, cmdMeta)
	} else {
		// automatically turn fields that begin with Flag into cli.Flag objects
		*flags = append(*flags, cmdMeta)
	}
	return
}

func parseMeta(prefix string, accesses []int, fieldType reflect.StructField, bo *BuildOptions) (cmdMeta commandMetadata, err error) {
	s := fieldType.Tag.Get("cli")

	cmdMeta.Skipped = false
	if s == "-" {
		cmdMeta.Skipped = true
		return cmdMeta, err
	}
	cmdMeta.Accesses = accesses
	cmdMeta.Required = false
	cmdMeta.Positional = false
	cmdMeta.UseShortOptions = false
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
		if section == "inline" {
			cmdMeta.Inline = true
			continue
		}
		if section == "required" {
			cmdMeta.Required = true
			requiredSetFromTags = true
			continue
		}
		if section == "shortOpt" {
			cmdMeta.UseShortOptions = true
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
			case "alias":
				cmdMeta.Aliases = strings.Split(keyValue[1], ",")
			case "hidden":
				cmdMeta.Hidden, err = strconv.ParseBool(keyValue[1])
				if err != nil {
					err = fmt.Errorf("failed to parse 'hidden' as a bool %s", err.Error())
				}
			case "default":
				cmdMeta.Default = new(string)
				*cmdMeta.Default = keyValue[1]
			case "entrypoint":
			case "shortOpt":
				cmdMeta.UseShortOptions, err = strconv.ParseBool(keyValue[1])
				if err != nil {
					err = fmt.Errorf("failed to parse 'shortOpt' as a bool %s", err.Error())
				}
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
		if !cmdMeta.Inline {
			cmdMeta.TypeInterface, err = flagType(fieldType)
			if err != nil {
				err = fmt.Errorf("cant find type for %s field: %s", cmdMeta.Name, err.Error())
				return
			}
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
	if prefix != "" {
		cmdMeta.Name = prefix + "-" + cmdMeta.Name
	}
	if cmdMeta.Name != "" {
		cmdMeta.Name = strcase.ToKebab(cmdMeta.Name)
	}
	if len(cmdMeta.Envs) == 0 {
		cmdMeta.Envs = []string{
			strcase.ToScreamingSnake(cmdMeta.Name),
		}
		if bo.EnvPrefix != "" {
			for i, v := range cmdMeta.Envs {
				cmdMeta.Envs[i] = bo.EnvPrefix + "_" + v
			}
		}
	}
	return cmdMeta, err
}
