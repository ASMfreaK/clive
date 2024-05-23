package clive

import (
	"encoding"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

type Counter struct {
	Value int
}

type TypePredicate interface {
	Predicate(reflect.Type) bool
}

type TypeFunctions interface {
	NewFlag(cmdMeta commandMetadata) (cli.Flag, error)
	SetValueFromString(val reflect.Value, s string) (err error)
	SetValueFromContext(value reflect.Value, flagName string, context *cli.Context) (err error)
	IsVariadic() bool
	SetValueFromStrings(val reflect.Value, s []string) (err error)
}

type TypeInterface interface {
	TypePredicate
	TypeFunctions
}

type (
	predicateHandler           func(fType reflect.Type) bool
	newFlagHandler             func(cmdMeta commandMetadata) (cli.Flag, error)
	setValueFromStringHandler  func(val reflect.Value, s string) (err error)
	setValueFromContextHandler func(value reflect.Value, flagName string, context *cli.Context) (err error)
	setValueFromStringsHandler func(val reflect.Value, s []string) (err error)
)

func Reflected[T any]() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

func typePredicate[T any](fType reflect.Type) bool {
	return fType == Reflected[T]()
}

func checkType[T any](fType reflect.Type) {
	expected := Reflected[T]()
	if fType != expected {
		panic(fmt.Errorf("wrong type: %s, expected: %s", fType.String(), expected.String()))
	}
}

func bits[T any]() int {
	rv := reflect.TypeOf((*T)(nil)).Elem()
	// if rv.Kind() == reflect.Int || rv.Kind() == reflect.Uint {
	// 	return 32
	// }
	return rv.Bits()
}

func convertSlice[U any, T any](ret *[]U, ts []T, conv func(*U, T) error) (err error) {
	*ret = make([]U, len(ts))
	for i, v := range ts {
		err = conv(&(*ret)[i], v)
		if err != nil {
			return
		}
	}
	return
}

func contextFunction[T any]() (ret func(ret *T, ctx *cli.Context, s string) error) {
	switch rv := interface{}(&ret).(type) {
	// scalars
	case *func(*int, *cli.Context, string) error:
		*rv = func(ret *int, ctx *cli.Context, s string) (err error) { *ret = ctx.Int(s); return }
	case *func(*int64, *cli.Context, string) error:
		*rv = func(ret *int64, ctx *cli.Context, s string) (err error) { *ret = ctx.Int64(s); return }
	case *func(*uint, *cli.Context, string) error:
		*rv = func(ret *uint, ctx *cli.Context, s string) (err error) { *ret = ctx.Uint(s); return }
	case *func(*uint64, *cli.Context, string) error:
		*rv = func(ret *uint64, ctx *cli.Context, s string) (err error) { *ret = ctx.Uint64(s); return }
	case *func(*float32, *cli.Context, string) error:
		*rv = func(ret *float32, ctx *cli.Context, s string) (err error) { *ret = float32(ctx.Float64(s)); return }
	case *func(*float64, *cli.Context, string) error:
		*rv = func(ret *float64, ctx *cli.Context, s string) (err error) { *ret = ctx.Float64(s); return }
	case *func(*string, *cli.Context, string) error:
		*rv = func(ret *string, ctx *cli.Context, s string) (err error) { *ret = ctx.String(s); return }
	case *func(*time.Duration, *cli.Context, string) error:
		*rv = func(ret *time.Duration, ctx *cli.Context, s string) (err error) { *ret = ctx.Duration(s); return }
	case *func(*bool, *cli.Context, string) error:
		*rv = func(ret *bool, ctx *cli.Context, s string) (err error) { *ret = ctx.Bool(s); return }
	case *func(*Counter, *cli.Context, string) error:
		*rv = func(ret *Counter, ctx *cli.Context, s string) (err error) { ret.Value = ctx.Count(s); return }
	// slices
	case *func(*[]int, *cli.Context, string) error:
		*rv = func(ret *[]int, ctx *cli.Context, s string) (err error) { *ret = ctx.IntSlice(s); return }
	case *func(*[]int64, *cli.Context, string) error:
		*rv = func(ret *[]int64, ctx *cli.Context, s string) (err error) { *ret = ctx.Int64Slice(s); return }
	case *func(*[]uint, *cli.Context, string) error:
		*rv = func(ret *[]uint, ctx *cli.Context, s string) (err error) { *ret = ctx.UintSlice(s); return }
	case *func(*[]uint64, *cli.Context, string) error:
		*rv = func(ret *[]uint64, ctx *cli.Context, s string) (err error) { *ret = ctx.Uint64Slice(s); return }
	case *func(*[]float32, *cli.Context, string) error:
		*rv = func(ret *[]float32, ctx *cli.Context, s string) (err error) {
			return convertSlice[float32, float64](
				ret, ctx.Float64Slice(s),
				func(f1 *float32, f2 float64) error { *f1 = float32(f2); return nil })
		}
	case *func(*[]float64, *cli.Context, string) error:
		*rv = func(ret *[]float64, ctx *cli.Context, s string) (err error) { *ret = ctx.Float64Slice(s); return }
	case *func(*[]string, *cli.Context, string) error:
		*rv = func(ret *[]string, ctx *cli.Context, s string) (err error) { *ret = ctx.StringSlice(s); return }
	case *func(*[]time.Duration, *cli.Context, string) error:
		*rv = func(ret *[]time.Duration, ctx *cli.Context, s string) (err error) {
			return convertSlice[time.Duration, string](ret, ctx.StringSlice(s), parseStandartTypes[time.Duration])
		}
	case *func(*[]bool, *cli.Context, string) error:
		*rv = func(ret *[]bool, ctx *cli.Context, s string) (err error) {
			return convertSlice[bool, string](ret, ctx.StringSlice(s), parseStandartTypes[bool])
		}
	default:
		panic("unexpected type " + reflect.TypeOf((*T)(nil)).Elem().String())
	}
	return
}

func parseStandartTypes[T any](ret *T, s string) (err error) {
	if isVariadic[T]() {
		err = parseStandartSliceTypes[T](ret, strings.Split(s, ","))
		return
	}
	switch rv := interface{}(ret).(type) {
	case *int:
		var def int64
		def, err = strconv.ParseInt(s, 0, bits[T]())
		if err != nil {
			return
		}
		*rv = int(def)
	case *int64:
		*rv, err = strconv.ParseInt(s, 0, bits[T]())
	case *uint:
		var def uint64
		def, err = strconv.ParseUint(s, 0, bits[T]())
		if err != nil {
			return
		}
		*rv = uint(def)
	case *uint64:
		*rv, err = strconv.ParseUint(s, 0, bits[T]())
	case *float32:
		var def float64
		def, err = strconv.ParseFloat(s, bits[T]())
		if err != nil {
			return
		}
		*rv = float32(def)
	case *float64:
		*rv, err = strconv.ParseFloat(s, bits[T]())
	case *string:
		*rv = s
	case *time.Duration:
		*rv, err = time.ParseDuration(s)
	case *bool:
		*rv, err = strconv.ParseBool(s)
	case *Counter:
		var def int64
		def, err = strconv.ParseInt(s, 0, bits[int]())
		if err != nil {
			return
		}
		rv.Value = int(def)
	// case encoding.TextUnmarshaler:
	// 	err = rv.UnmarshalText([]byte(s))

	// case *[]encoding.TextUnmarshaler:
	// 	*rv, err = parseSlice[[]encoding.TextUnmarshaler](strings.Split(s, ","))
	default:
		err = fmt.Errorf("unexpected type in  parseStandartTypes" + reflect.TypeOf((*T)(nil)).Elem().String())
	}
	return
}

func parseSlice[T []U, U any](ret *[]U, s []string) (err error) {
	return convertSlice[U, string](ret, s, parseStandartTypes[U])
}

func cliSliceFromStandartSliceTypes[T any](val *T) (ret reflect.Value, err error) {
	switch rv := interface{}(val).(type) {
	case *[]int:
		ret = reflect.ValueOf(cli.NewIntSlice((*rv)...))
	case *[]int64:
		ret = reflect.ValueOf(cli.NewInt64Slice((*rv)...))
	case *[]uint:
		ret = reflect.ValueOf(cli.NewUintSlice((*rv)...))
	case *[]uint64:
		ret = reflect.ValueOf(cli.NewUint64Slice((*rv)...))
	case *[]float32:
		var realSlice []float64
		convertSlice[float64, float32](&realSlice, *rv, func(f1 *float64, f2 float32) error { *f1 = float64(f2); return nil })
		ret = reflect.ValueOf(cli.NewFloat64Slice(realSlice...))
	case *[]float64:
		ret = reflect.ValueOf(cli.NewFloat64Slice((*rv)...))
	case *[]string:
		ret = reflect.ValueOf(cli.NewStringSlice((*rv)...))
	case *[]time.Duration:
		var realSlice []string
		convertSlice[string, time.Duration](&realSlice, *rv, func(s *string, d time.Duration) error { *s = d.String(); return nil })
		ret = reflect.ValueOf(cli.NewStringSlice(realSlice...))
	case *[]bool:
		var realSlice []string
		convertSlice[string, bool](&realSlice, *rv, func(s *string, b bool) error { *s = strconv.FormatBool(b); return nil })
		ret = reflect.ValueOf(cli.NewStringSlice(realSlice...))
	default:
		err = fmt.Errorf("unexpected type in  parseStandartTypes" + reflect.TypeOf((*T)(nil)).Elem().String())
	}
	return
}

func parseStandartSliceTypes[T any](ret *T, s []string) (err error) {
	switch rv := interface{}(ret).(type) {
	case *[]int:
		err = parseSlice[[]int](rv, s)
	case *[]int64:
		err = parseSlice[[]int64](rv, s)
	case *[]uint:
		err = parseSlice[[]uint](rv, s)
	case *[]uint64:
		err = parseSlice[[]uint64](rv, s)
	case *[]float32:
		err = parseSlice[[]float32](rv, s)
	case *[]float64:
		err = parseSlice[[]float64](rv, s)
	case *[]string:
		err = parseSlice[[]string](rv, s)
	case *[]time.Duration:
		err = parseSlice[[]time.Duration](rv, s)
	case *[]bool:
		err = parseSlice[[]bool](rv, s)
	default:
		err = fmt.Errorf("unexpected type in parseStandartSliceTypes" + reflect.TypeOf((*T)(nil)).Elem().String())
	}
	return
}

func isVariadic[T any]() bool {
	return Reflected[T]().Kind() == reflect.Slice
}

func setValueFromString[T any](val reflect.Value, s string) (err error) {
	checkType[*T](val.Type())
	err = parseStandartTypes[T](val.Interface().(*T), s)
	if err != nil {
		return
	}
	return
}

func setValueFromStrings[T any](val reflect.Value, s []string) (err error) {
	checkType[*T](val.Type())
	err = parseStandartSliceTypes[T](val.Interface().(*T), s)
	if err != nil {
		return
	}
	return
}

func setValueFromContext[T any](value reflect.Value, flagName string, context *cli.Context) (err error) {
	checkType[*T](value.Type())
	err = contextFunction[T]()(value.Interface().(*T), context, flagName)
	if err != nil {
		return
	}
	return
}

func newFlag[T any, Flag any](cmdMeta commandMetadata) (flag cli.Flag, err error) {
	var def T
	var defRefPtr reflect.Value
	if cmdMeta.Default != nil {
		defRefPtr = reflect.ValueOf(&def)
		err = setValueFromString[T](defRefPtr, *cmdMeta.Default)
		if err != nil {
			return
		}
		if Reflected[T]() == Reflected[float32]() {
			def64 := float64(*defRefPtr.Interface().(*float32))
			defRefPtr = reflect.ValueOf(&def64)
		}
		if isVariadic[T]() {
			defRefPtr, err = cliSliceFromStandartSliceTypes[T](&def)
			if err != nil {
				return
			}
		} else {
			defRefPtr = defRefPtr.Elem()
		}
	}
	typedFlag := new(Flag)
	refTypedFlag := reflect.ValueOf(typedFlag)
	refTypedFlag.Elem().FieldByName("Name").SetString(cmdMeta.Name)
	refTypedFlag.Elem().FieldByName("EnvVars").Set(reflect.ValueOf(cmdMeta.Envs))
	refTypedFlag.Elem().FieldByName("Aliases").Set(reflect.ValueOf(cmdMeta.Aliases))
	if defRefPtr.IsValid() {
		refTypedFlag.Elem().FieldByName("Value").Set(defRefPtr)
	}
	refTypedFlag.Elem().FieldByName("Hidden").SetBool(cmdMeta.Hidden)
	refTypedFlag.Elem().FieldByName("Usage").SetString(cmdMeta.Usage)

	if Reflected[T]() == Reflected[Counter]() {
		refTypedFlag.Elem().FieldByName("Count").Set(reflect.New(Reflected[int]()))
	}

	flag = refTypedFlag.Interface().(cli.Flag)
	return
}

type StandardType struct {
	predicate           predicateHandler
	newFlag             newFlagHandler
	setValueFromString  setValueFromStringHandler
	setValueFromContext setValueFromContextHandler
	setValueFromStrings setValueFromStringsHandler
}

func NewStandardType[T any, Flag any]() *StandardType {
	var variadic setValueFromStringsHandler = nil
	if isVariadic[T]() {
		variadic = setValueFromStrings[T]
	}
	return &StandardType{
		predicate:           typePredicate[T],
		newFlag:             newFlag[T, Flag],
		setValueFromString:  setValueFromString[T],
		setValueFromContext: setValueFromContext[T],
		setValueFromStrings: variadic,
	}
}

func (nt *StandardType) Predicate(fType reflect.Type) bool {
	return nt.predicate(fType)
}

func (nt *StandardType) SetValueFromString(val reflect.Value, s string) (err error) {
	return nt.setValueFromString(val, s)
}
func (nt *StandardType) NewFlag(cmdMeta commandMetadata) (cli.Flag, error) {
	return nt.newFlag(cmdMeta)
}
func (nt *StandardType) SetValueFromContext(value reflect.Value, flagName string, context *cli.Context) error {
	return nt.setValueFromContext(value, flagName, context)
}

func (nt *StandardType) IsVariadic() bool {
	return nt.setValueFromStrings != nil
}
func (nt *StandardType) SetValueFromStrings(val reflect.Value, s []string) (err error) {
	return nt.setValueFromStrings(val, s)
}

type InterfaceType struct {
	interfaceType reflect.Type
	under         TypeInterface
	underType     reflect.Type

	convert func(convertInto reflect.Value, fromUnderType reflect.Value) error
}

func (ifaceType *InterfaceType) Predicate(fType reflect.Type) bool {
	if ifaceType.IsVariadic() {
		if fType.Kind() != reflect.Slice {
			return false
		}
		fType = fType.Elem()
	}
	return reflect.PointerTo(fType).Implements(ifaceType.interfaceType)
}

func (ifaceType *InterfaceType) NewFlag(cmdMeta commandMetadata) (cli.Flag, error) {
	return ifaceType.under.NewFlag(cmdMeta)
}

func (ifaceType *InterfaceType) SetValueFromString(value reflect.Value, s string) (err error) {
	if value.Kind() != reflect.Ptr {
		err = fmt.Errorf("expected pointer, got %s", value.Type().String())
		return
	}
	underVal := reflect.New(ifaceType.underType)
	err = ifaceType.under.SetValueFromString(underVal, s)
	if err != nil {
		return
	}
	return ifaceType.convert(value, underVal)
}

func (ifaceType *InterfaceType) SetValueFromContext(value reflect.Value, flagName string, context *cli.Context) (err error) {
	if value.Kind() != reflect.Ptr {
		err = fmt.Errorf("expected pointer, got %s", value.Type().String())
		return
	}
	underVal := reflect.New(ifaceType.underType)
	err = ifaceType.under.SetValueFromContext(underVal, flagName, context)
	if err != nil {
		return
	}
	return ifaceType.convert(value, underVal)
}
func (ifaceType *InterfaceType) IsVariadic() bool { return ifaceType.under.IsVariadic() }
func (ifaceType *InterfaceType) SetValueFromStrings(value reflect.Value, s []string) (err error) {
	if value.Kind() != reflect.Ptr {
		err = fmt.Errorf("expected pointer, got %s", value.Type().String())
		return
	}
	underVal := reflect.New(ifaceType.underType)
	err = ifaceType.under.SetValueFromStrings(underVal, s)
	if err != nil {
		return
	}
	return ifaceType.convert(value, underVal)
}

type PointerTo struct {
	ti TypeInterface
}

func (ptrTo *PointerTo) maybeInitializeDereference(value reflect.Value) reflect.Value {
	elem := value.Elem()
	if value.Kind() != reflect.Ptr || elem.Kind() != reflect.Ptr {
		panic("Wrong kind in pointer to: " + elem.Type().String())
	}
	if elem.IsNil() {
		elem.Set(reflect.New(elem.Type().Elem()))
	}
	return elem
}

func (ptrTo *PointerTo) Predicate(fType reflect.Type) bool {
	return fType.Kind() == reflect.Ptr
}
func (ptrTo *PointerTo) SetValueFromString(value reflect.Value, s string) (err error) {
	return ptrTo.ti.SetValueFromString(ptrTo.maybeInitializeDereference(value), s)
}
func (ptrTo *PointerTo) NewFlag(cmdMeta commandMetadata) (cli.Flag, error) {
	return ptrTo.ti.NewFlag(cmdMeta)
}
func (ptrTo *PointerTo) SetValueFromContext(value reflect.Value, flagName string, context *cli.Context) error {
	return ptrTo.ti.SetValueFromContext(ptrTo.maybeInitializeDereference(value), flagName, context)
}
func (ptrTo *PointerTo) IsVariadic() bool { return ptrTo.ti.IsVariadic() }
func (ptrTo *PointerTo) SetValueFromStrings(value reflect.Value, s []string) (err error) {
	return ptrTo.ti.SetValueFromStrings(ptrTo.maybeInitializeDereference(value), s)
}

func flagType(fieldType reflect.StructField) (TypeInterface, error) {
	fieldValueType := fieldType.Type
	var ptrTo *PointerTo = nil
	var ptrCurrent *PointerTo = nil
	for ptrCurrent.Predicate(fieldValueType) {
		if ptrTo == nil {
			ptrTo = &PointerTo{}
			ptrCurrent = ptrTo
		} else {
			newPtrTo := &PointerTo{}
			ptrCurrent.ti = newPtrTo
			ptrCurrent = newPtrTo
		}
		fieldValueType = fieldValueType.Elem()
	}
	for _, t := range types {
		if t.Predicate(fieldValueType) {
			if ptrCurrent == nil {
				return t, nil
			} else {
				ptrCurrent.ti = t
				return ptrTo, nil
			}
		}
	}
	return nil, fmt.Errorf("unsupported flag generator type: %s", fieldType.Type.String())
}

func genericSliceConvert(convertInto, from reflect.Value, convertOne func(reflect.Value, reflect.Value) error) (err error) {
	if convertInto.Kind() != reflect.Pointer || convertInto.Elem().Kind() != reflect.Slice {
		err = fmt.Errorf("unsupported type for convertInto in genericSliceConvert: %s", convertInto.Type().String())
		return
	}
	if from.Kind() != reflect.Pointer || from.Elem().Kind() != reflect.Slice {
		err = fmt.Errorf("unsupported type for from in genericSliceConvert: %s", from.Type().String())
		return
	}
	length := from.Elem().Len()
	slice := reflect.MakeSlice(
		convertInto.Type().Elem(),
		length,
		length,
	)
	for i := 0; i < length; i++ {
		err = convertOne(slice.Index(i).Addr(), from.Elem().Index(i).Addr())
		if err != nil {
			return
		}
	}
	convertInto.Elem().Set(slice)
	return
}

func genericConvertTextUnmarshal(convertInto, fromUnderType reflect.Value) error {
	return convertInto.Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(fromUnderType.Elem().String()))
}

var types = []TypeInterface{
	NewStandardType[int, cli.IntFlag](),
	NewStandardType[int64, cli.Int64Flag](),
	NewStandardType[uint, cli.UintFlag](),
	NewStandardType[uint64, cli.Uint64Flag](),
	NewStandardType[float32, cli.Float64Flag](),
	NewStandardType[float64, cli.Float64Flag](),
	NewStandardType[string, cli.StringFlag](),
	NewStandardType[time.Duration, cli.DurationFlag](),
	NewStandardType[bool, cli.BoolFlag](),
	NewStandardType[[]int, cli.IntSliceFlag](),
	NewStandardType[[]int64, cli.Int64SliceFlag](),
	NewStandardType[[]uint, cli.UintSliceFlag](),
	NewStandardType[[]uint64, cli.Uint64SliceFlag](),
	NewStandardType[[]float32, cli.Float64SliceFlag](),
	NewStandardType[[]float64, cli.Float64SliceFlag](),
	NewStandardType[[]string, cli.StringSliceFlag](),
	NewStandardType[[]time.Duration, cli.StringSliceFlag](),
	NewStandardType[[]bool, cli.StringSliceFlag](),
	NewStandardType[Counter, cli.BoolFlag](),
	&InterfaceType{
		interfaceType: Reflected[encoding.TextUnmarshaler](),

		under:     NewStandardType[string, cli.StringFlag](),
		underType: Reflected[string](),

		convert: genericConvertTextUnmarshal,
	},
	&InterfaceType{
		interfaceType: Reflected[encoding.TextUnmarshaler](),

		under:     NewStandardType[[]string, cli.StringSliceFlag](),
		underType: Reflected[[]string](),

		convert: func(convertInto, fromUnderType reflect.Value) error {
			return genericSliceConvert(convertInto, fromUnderType, genericConvertTextUnmarshal)
		},
	},
}
