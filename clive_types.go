package clive

import (
	"encoding"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

type TypePredicate interface {
	Predicate(reflect.Type) bool
}

type TypeFunctions interface {
	NewFlag(cmdmeta commandMetadata) (cli.Flag, error)
	SetValueFromString(val reflect.Value, s string) (err error)
	SetValueFromContext(value reflect.Value, flagName string, context *cli.Context) (err error)
	IsVariadic() bool
	SetValueFromStrings(val reflect.Value, s []string) (err error)
}

type TypeInterface interface {
	TypePredicate
	TypeFunctions
}

type functions struct {
	newFlag             func(f *functions, cmdmeta commandMetadata) (cli.Flag, error)
	setValueFromString  func(val reflect.Value, s string) (err error)
	setValueFromContext func(value reflect.Value, flagName string, context *cli.Context) (err error)
}

func (nt *functions) SetValueFromString(val reflect.Value, s string) (err error) {
	return nt.setValueFromString(val, s)
}
func (nt *functions) NewFlag(cmdmeta commandMetadata) (cli.Flag, error) {
	return nt.newFlag(nt, cmdmeta)
}
func (nt *functions) SetValueFromContext(value reflect.Value, flagName string, context *cli.Context) error {
	return nt.setValueFromContext(value, flagName, context)
}

type NamedType struct {
	*functions
	Name     string
	Variadic func(val reflect.Value, s []string) (err error)
}

func (nt *NamedType) Predicate(fType reflect.Type) bool {
	return fType.String() == nt.Name
}

func (nt *NamedType) IsVariadic() bool {
	return nt.Variadic != nil
}
func (nt *NamedType) SetValueFromStrings(val reflect.Value, s []string) (err error) {
	return nt.Variadic(val, s)
}

type InterfaceType struct {
	*functions
	Type reflect.Type
}

func (nt *InterfaceType) Predicate(fType reflect.Type) bool {
	return reflect.PointerTo(fType).Implements(nt.Type)
}
func (nt *InterfaceType) IsVariadic() bool { return false }
func (nt *InterfaceType) SetValueFromStrings(val reflect.Value, s []string) (err error) {
	panic("not implemented")
}

func checkType(fType reflect.Type, name string) {
	fTypeS := fType.String()
	if fTypeS != name {
		panic(fmt.Sprintf("wrong type in setValueFromString: %s, expected: %s", fTypeS, name))
	}
}

func flagType(fieldType reflect.StructField) (TypeInterface, error) {
	for _, t := range types {
		if t.Predicate(fieldType.Type) {
			return t, nil
		}
	}
	return nil, errors.Errorf("unsupported flag generator type: %s", fieldType.Type.String())
}

var types = []TypeInterface{
	&NamedType{
		Name: "int",
		functions: &functions{
			newFlag: func(f *functions, cmdmeta commandMetadata) (flag cli.Flag, err error) {
				var def int
				if cmdmeta.Default != "" {
					err = f.setValueFromString(reflect.ValueOf(&def), cmdmeta.Default)
					if err != nil {
						return
					}
				}
				flag = &cli.IntFlag{
					Name:    cmdmeta.Name,
					EnvVars: cmdmeta.Envs,
					Value:   def,
					Hidden:  cmdmeta.Hidden,
					Usage:   cmdmeta.Usage,
					// Action: func(ctx *cli.Context, i int) error {
					// 	return nil
					// },
				}
				return
			},
			setValueFromString: func(val reflect.Value, s string) (err error) {
				checkType(val.Type(), "*int")
				def, err := strconv.ParseInt(s, 10, 64)
				if err != nil {
					return
				}
				val.Elem().Set(reflect.ValueOf(int(def)))
				return
			},
			setValueFromContext: func(value reflect.Value, flagName string, context *cli.Context) error {
				value.Elem().SetInt(int64(context.Int(flagName)))
				return nil
			},
		},
	},
	&NamedType{
		Name: "int64",
		functions: &functions{
			setValueFromString: func(val reflect.Value, s string) (err error) {
				checkType(val.Type(), "*int64")
				def, err := strconv.ParseInt(s, 10, 64)
				if err != nil {
					return
				}
				val.Elem().Set(reflect.ValueOf(int64(def)))
				return
			},
			newFlag: func(f *functions, cmdmeta commandMetadata) (flag cli.Flag, err error) {
				var def int64
				if cmdmeta.Default != "" {
					err = f.setValueFromString(reflect.ValueOf(&def), cmdmeta.Default)
					if err != nil {
						return
					}
				}
				flag = &cli.Int64Flag{
					Name:    cmdmeta.Name,
					EnvVars: cmdmeta.Envs,
					Value:   def,
					Hidden:  cmdmeta.Hidden,
					Usage:   cmdmeta.Usage,
				}
				return
			},
			setValueFromContext: func(value reflect.Value, flagName string, context *cli.Context) error {
				value.Elem().SetInt(context.Int64(flagName))
				return nil
			},
		},
	},
	&NamedType{
		Name: "uint",
		functions: &functions{
			setValueFromString: func(val reflect.Value, s string) (err error) {
				checkType(val.Type(), "*uint")
				def, err := strconv.ParseUint(s, 10, 64)
				if err != nil {
					return
				}
				val.Elem().Set(reflect.ValueOf(uint(def)))
				return
			},
			newFlag: func(f *functions, cmdmeta commandMetadata) (flag cli.Flag, err error) {
				var def uint
				if cmdmeta.Default != "" {
					err = f.setValueFromString(reflect.ValueOf(&def), cmdmeta.Default)
					if err != nil {
						return
					}
				}
				flag = &cli.UintFlag{
					Name:    cmdmeta.Name,
					EnvVars: cmdmeta.Envs,
					Value:   def,
					Hidden:  cmdmeta.Hidden,
					Usage:   cmdmeta.Usage,
				}
				return
			},
			setValueFromContext: func(value reflect.Value, flagName string, context *cli.Context) error {
				value.Elem().SetUint(uint64(context.Uint(flagName)))
				return nil
			},
		},
	},
	&NamedType{
		Name: "uint64",
		functions: &functions{
			setValueFromString: func(val reflect.Value, s string) (err error) {
				checkType(val.Type(), "*uint64")
				def, err := strconv.ParseUint(s, 10, 64)
				if err != nil {
					return
				}
				val.Elem().Set(reflect.ValueOf(uint64(def)))
				return
			},
			newFlag: func(f *functions, cmdmeta commandMetadata) (flag cli.Flag, err error) {
				var def uint64
				if cmdmeta.Default != "" {
					err = f.setValueFromString(reflect.ValueOf(&def), cmdmeta.Default)
					if err != nil {
						return
					}
				}
				flag = &cli.Uint64Flag{
					Name:    cmdmeta.Name,
					EnvVars: cmdmeta.Envs,
					Value:   def,
					Hidden:  cmdmeta.Hidden,
					Usage:   cmdmeta.Usage,
				}
				return
			},
			setValueFromContext: func(value reflect.Value, flagName string, context *cli.Context) error {
				value.Elem().SetUint(context.Uint64(flagName))
				return nil
			},
		},
	},

	&NamedType{
		Name: "float32",
		functions: &functions{
			setValueFromString: func(val reflect.Value, s string) (err error) {
				checkType(val.Type(), "*float32")
				def, err := strconv.ParseFloat(s, 32)
				if err != nil {
					return
				}
				val.Elem().Set(reflect.ValueOf(float32(def)))
				return
			},
			newFlag: func(f *functions, cmdmeta commandMetadata) (flag cli.Flag, err error) {
				var def float32
				if cmdmeta.Default != "" {
					err = f.setValueFromString(reflect.ValueOf(&def), cmdmeta.Default)
					if err != nil {
						return
					}
				}
				flag = &cli.Float64Flag{
					Name:    cmdmeta.Name,
					EnvVars: cmdmeta.Envs,
					Value:   float64(def),
					Hidden:  cmdmeta.Hidden,
					Usage:   cmdmeta.Usage,
				}
				return
			},
			setValueFromContext: func(value reflect.Value, flagName string, context *cli.Context) error {
				value.Elem().SetFloat(context.Float64(flagName))
				return nil
			},
		},
	},

	&NamedType{
		Name: "float64",
		functions: &functions{
			setValueFromString: func(val reflect.Value, s string) (err error) {
				checkType(val.Type(), "*float64")
				def, err := strconv.ParseFloat(s, 64)
				if err != nil {
					return
				}
				val.Elem().Set(reflect.ValueOf(float64(def)))
				return
			},
			newFlag: func(f *functions, cmdmeta commandMetadata) (flag cli.Flag, err error) {
				var def float64
				if cmdmeta.Default != "" {
					err = f.setValueFromString(reflect.ValueOf(&def), cmdmeta.Default)
					if err != nil {
						return
					}
				}
				flag = &cli.Float64Flag{
					Name:    cmdmeta.Name,
					EnvVars: cmdmeta.Envs,
					Value:   def,
					Hidden:  cmdmeta.Hidden,
					Usage:   cmdmeta.Usage,
				}
				return
			},
			setValueFromContext: func(value reflect.Value, flagName string, context *cli.Context) error {
				value.Elem().SetFloat(context.Float64(flagName))
				return nil
			},
		},
	},
	&NamedType{
		Name: "bool",
		functions: &functions{
			setValueFromString: func(val reflect.Value, s string) (err error) {
				checkType(val.Type(), "*bool")
				def, err := strconv.ParseBool(s)
				if err != nil {
					return
				}
				val.Elem().Set(reflect.ValueOf(bool(def)))
				return
			},
			newFlag: func(f *functions, cmdmeta commandMetadata) (flag cli.Flag, err error) {
				var def bool
				if cmdmeta.Default != "" {
					err = f.setValueFromString(reflect.ValueOf(&def), cmdmeta.Default)
					if err != nil {
						return
					}
				}
				if !def {
					flag = &cli.BoolFlag{
						Name:    cmdmeta.Name,
						EnvVars: cmdmeta.Envs,
						Hidden:  cmdmeta.Hidden,
						Usage:   cmdmeta.Usage,
					}
				} else {
					flag = &cli.BoolFlag{
						Name:    cmdmeta.Name,
						EnvVars: cmdmeta.Envs,
						Value:   true,
						Hidden:  cmdmeta.Hidden,
						Usage:   cmdmeta.Usage,
					}
				}
				return
			},
			setValueFromContext: func(value reflect.Value, flagName string, context *cli.Context) error {
				value.Elem().SetBool(context.Bool(flagName))
				return nil
			},
		},
	},

	&NamedType{
		Name: "string",
		functions: &functions{
			setValueFromString: func(val reflect.Value, s string) (err error) {
				checkType(val.Type(), "*string")
				val.Elem().Set(reflect.ValueOf(s))
				return
			},
			newFlag: func(f *functions, cmdmeta commandMetadata) (flag cli.Flag, err error) {
				flag = &cli.StringFlag{
					Name:    cmdmeta.Name,
					EnvVars: cmdmeta.Envs,
					Value:   cmdmeta.Default,
					Hidden:  cmdmeta.Hidden,
					Usage:   cmdmeta.Usage,
				}
				return
			},
			setValueFromContext: func(value reflect.Value, flagName string, context *cli.Context) error {
				value.Elem().SetString(context.String(flagName))
				return nil
			},
		},
	},
	&NamedType{
		Name: "time.Duration",
		functions: &functions{
			setValueFromString: func(val reflect.Value, s string) (err error) {
				checkType(val.Type(), "*time.Duration")
				def, err := time.ParseDuration(s)
				if err != nil {
					return
				}
				val.Elem().Set(reflect.ValueOf(def))
				return
			},
			newFlag: func(f *functions, cmdmeta commandMetadata) (flag cli.Flag, err error) {
				var def time.Duration
				if cmdmeta.Default != "" {
					err = f.setValueFromString(reflect.ValueOf(&def), cmdmeta.Default)
					if err != nil {
						return
					}
				}
				flag = &cli.DurationFlag{
					Name:    cmdmeta.Name,
					EnvVars: cmdmeta.Envs,
					Value:   def,
					Hidden:  cmdmeta.Hidden,
					Usage:   cmdmeta.Usage,
				}
				return
			},
			setValueFromContext: func(value reflect.Value, flagName string, context *cli.Context) error {
				value.Elem().SetInt(context.Duration(flagName).Nanoseconds())
				return nil
			},
		},
	},
	&NamedType{
		Name: "[]int",
		Variadic: func(val reflect.Value, s []string) (err error) {
			checkType(val.Type(), "*[]int")
			defs := make([]int, 0, len(s))
			for _, s := range s {
				var d int64
				d, err = strconv.ParseInt(s, 10, 32)
				if err != nil {
					return
				}
				defs = append(defs, int(d))
			}
			val.Elem().Set(reflect.ValueOf(defs))
			return
		},
		functions: &functions{
			setValueFromString: func(val reflect.Value, s string) (err error) {
				checkType(val.Type(), "*[]int")
				split := strings.Split(s, ",")
				defs := make([]int, 0, len(split))
				for _, s := range split {
					var d int64
					d, err = strconv.ParseInt(s, 10, 32)
					if err != nil {
						return
					}
					defs = append(defs, int(d))
				}
				val.Elem().Set(reflect.ValueOf(defs))
				return
			},
			newFlag: func(f *functions, cmdmeta commandMetadata) (flag cli.Flag, err error) {
				var def *cli.IntSlice
				if cmdmeta.Default != "" {
					var defSlice []int
					err = f.setValueFromString(reflect.ValueOf(&defSlice), cmdmeta.Default)
					if err != nil {
						return
					}
					def = cli.NewIntSlice(defSlice...)
				}
				flag = &cli.IntSliceFlag{
					Name:    cmdmeta.Name,
					EnvVars: cmdmeta.Envs,
					Value:   def,
					Hidden:  cmdmeta.Hidden,
					Usage:   cmdmeta.Usage,
				}
				return
			},
			setValueFromContext: func(value reflect.Value, flagName string, context *cli.Context) error {
				value.Elem().Set(genericSliceOf(context.IntSlice(flagName)))
				return nil
			},
		},
	},

	&NamedType{
		Name: "[]int64",
		Variadic: func(val reflect.Value, s []string) (err error) {
			checkType(val.Type(), "*[]int64")
			defs := make([]int64, 0, len(s))
			for _, s := range s {
				d, _ := strconv.ParseInt(s, 10, 64)
				defs = append(defs, int64(d))
			}
			val.Elem().Set(reflect.ValueOf(defs))
			return
		},
		functions: &functions{
			setValueFromString: func(val reflect.Value, s string) (err error) {
				checkType(val.Type(), "*[]int64")
				split := strings.Split(s, ",")
				defs := make([]int64, 0, len(split))
				for _, s := range split {
					d, _ := strconv.ParseInt(s, 10, 64)
					defs = append(defs, int64(d))
				}
				val.Elem().Set(reflect.ValueOf(defs))
				return
			},
			newFlag: func(f *functions, cmdmeta commandMetadata) (flag cli.Flag, err error) {
				var def *cli.Int64Slice
				if cmdmeta.Default != "" {
					var defSlice []int64
					err = f.setValueFromString(reflect.ValueOf(&defSlice), cmdmeta.Default)
					if err != nil {
						return
					}
					def = cli.NewInt64Slice(defSlice...)
				}
				flag = &cli.Int64SliceFlag{
					Name:    cmdmeta.Name,
					EnvVars: cmdmeta.Envs,
					Value:   def,
					Hidden:  cmdmeta.Hidden,
					Usage:   cmdmeta.Usage,
				}
				return
			},
			setValueFromContext: func(value reflect.Value, flagName string, context *cli.Context) error {
				value.Elem().Set(genericSliceOf(context.Int64Slice(flagName)))
				return nil
			},
		},
	},

	// urfave/cli does not have unsigned types yet
	&NamedType{
		Name: "[]uint",
		Variadic: func(val reflect.Value, s []string) (err error) {
			checkType(val.Type(), "*[]uint")
			defs := make([]uint, 0, len(s))
			for _, s := range s {
				var d uint64
				d, err = strconv.ParseUint(s, 10, 32)
				if err != nil {
					return
				}
				defs = append(defs, uint(d))
			}
			val.Elem().Set(reflect.ValueOf(defs))
			return
		},
		functions: &functions{
			setValueFromString: func(val reflect.Value, s string) (err error) {
				checkType(val.Type(), "*[]uint")
				split := strings.Split(s, ",")
				defs := make([]uint, 0, len(split))
				for _, s := range split {
					var d uint64
					d, err = strconv.ParseUint(s, 10, 32)
					if err != nil {
						return
					}
					defs = append(defs, uint(d))
				}
				val.Elem().Set(reflect.ValueOf(defs))
				return
			},
			newFlag: func(f *functions, cmdmeta commandMetadata) (flag cli.Flag, err error) {
				var def *cli.UintSlice
				if cmdmeta.Default != "" {
					var defSlice []uint
					err = f.setValueFromString(reflect.ValueOf(&defSlice), cmdmeta.Default)
					if err != nil {
						return
					}
					def = cli.NewUintSlice(defSlice...)
				}
				flag = &cli.UintSliceFlag{
					Name:    cmdmeta.Name,
					EnvVars: cmdmeta.Envs,
					Hidden:  cmdmeta.Hidden,
					Usage:   cmdmeta.Usage,
					Value:   def,
				}
				return
			},
			setValueFromContext: func(value reflect.Value, flagName string, context *cli.Context) error {
				value.Elem().Set(genericSliceOf(context.UintSlice(flagName)))
				return nil
			},
		},
	},
	&NamedType{
		Name: "[]uint64",
		Variadic: func(val reflect.Value, s []string) (err error) {
			checkType(val.Type(), "*[]uint64")
			defs := make([]uint64, 0, len(s))
			for _, s := range s {
				var d uint64
				d, err = strconv.ParseUint(s, 10, 64)
				if err != nil {
					return
				}
				defs = append(defs, uint64(d))
			}
			val.Elem().Set(reflect.ValueOf(defs))
			return
		},
		functions: &functions{
			setValueFromString: func(val reflect.Value, s string) (err error) {
				checkType(val.Type(), "*[]uint64")
				split := strings.Split(s, ",")
				defs := make([]uint64, 0, len(split))
				for _, s := range split {
					var d uint64
					d, err = strconv.ParseUint(s, 10, 64)
					if err != nil {
						return
					}
					defs = append(defs, uint64(d))
				}
				val.Elem().Set(reflect.ValueOf(defs))
				return
			},
			newFlag: func(f *functions, cmdmeta commandMetadata) (flag cli.Flag, err error) {
				var def *cli.Uint64Slice
				if cmdmeta.Default != "" {
					var defSlice []uint64
					err = f.setValueFromString(reflect.ValueOf(&defSlice), cmdmeta.Default)
					if err != nil {
						return
					}
					def = cli.NewUint64Slice(defSlice...)
				}
				flag = &cli.Uint64SliceFlag{
					Name:    cmdmeta.Name,
					EnvVars: cmdmeta.Envs,
					Hidden:  cmdmeta.Hidden,
					Usage:   cmdmeta.Usage,
					Value:   def,
				}
				return
			},
			setValueFromContext: func(value reflect.Value, flagName string, context *cli.Context) error {
				value.Elem().Set(genericSliceOf(context.Uint64Slice(flagName)))
				return nil
			},
		},
	},

	&NamedType{
		Name: "[]string",
		Variadic: func(val reflect.Value, s []string) (err error) {
			checkType(val.Type(), "*[]string")
			val.Elem().Set(reflect.ValueOf(s))
			return
		},
		functions: &functions{
			setValueFromString: func(val reflect.Value, s string) (err error) {
				checkType(val.Type(), "*[]string")
				val.Elem().Set(reflect.ValueOf(strings.Split(s, ",")))
				return
			},
			newFlag: func(f *functions, cmdmeta commandMetadata) (flag cli.Flag, err error) {
				var def *cli.StringSlice
				if cmdmeta.Default != "" {
					var defSlice []string
					err = f.setValueFromString(reflect.ValueOf(&defSlice), cmdmeta.Default)
					if err != nil {
						return
					}
					def = cli.NewStringSlice(defSlice...)
				}
				flag = &cli.StringSliceFlag{
					Name:    cmdmeta.Name,
					EnvVars: cmdmeta.Envs,
					Value:   def,
					Hidden:  cmdmeta.Hidden,
					Usage:   cmdmeta.Usage,
				}
				return
			},
			setValueFromContext: func(value reflect.Value, flagName string, context *cli.Context) error {
				value.Elem().Set(genericSliceOf(context.StringSlice(flagName)))
				return nil
			},
		},
	},
	&InterfaceType{
		Type: reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem(),
		functions: &functions{
			setValueFromString: func(val reflect.Value, s string) (err error) {
				// checkType(val.Type(), "*[]string")
				return val.Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(s))
			},
			newFlag: func(f *functions, cmdmeta commandMetadata) (flag cli.Flag, err error) {
				// var def *cli.StringSlice
				// if cmdmeta.Default != "" {
				// 	var defSlice []string
				// 	err = f.setValueFromString(reflect.ValueOf(&defSlice), cmdmeta.Default)
				// 	if err != nil {
				// 		return
				// 	}
				// 	def = cli.NewStringSlice(defSlice...)
				// }
				flag = &cli.StringFlag{
					Name:    cmdmeta.Name,
					EnvVars: cmdmeta.Envs,
					Value:   cmdmeta.Default,
					Hidden:  cmdmeta.Hidden,
					Usage:   cmdmeta.Usage,
				}
				return
			},
			setValueFromContext: func(value reflect.Value, flagName string, context *cli.Context) error {
				return value.Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(context.String(flagName)))
			},
		},
	},
}
