package mapping

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"lazygo/core/stringx"
)

const (
	defaultOption   = "default"
	stringOption    = "string"
	optionalOption  = "optional"
	optionsOption   = "options"
	rangeOption     = "range"
	optionSeparator = "|"
	equalToken      = "="
)

var (
	errUnsupportedType = errors.New("unsupported type on setting field value")
	errNumberRange     = errors.New("wrong number range setting")
	optionsCache       = make(map[string]*optionsCacheValue)
	cacheLock          sync.RWMutex
)

var (
	errValueNotSettable = errors.New("value is not settable")
)

type optionsCacheValue struct {
	key     string
	options *fieldOptions
	err     error
}

func Deref(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return t
}

func Repr(v interface{}) string {
	if v == nil {
		return ""
	}

	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr && !val.IsNil() {
		val = val.Elem()
	}

	switch vt := val.Interface().(type) {
	case bool:
		return strconv.FormatBool(vt)
	case error:
		return vt.Error()
	case float32:
		return strconv.FormatFloat(float64(vt), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(vt, 'f', -1, 64)
	case fmt.Stringer:
		return vt.String()
	case int:
		return strconv.Itoa(vt)
	case int8:
		return strconv.Itoa(int(vt))
	case int16:
		return strconv.Itoa(int(vt))
	case int32:
		return strconv.Itoa(int(vt))
	case int64:
		return strconv.FormatInt(vt, 10)
	case string:
		return vt
	case uint:
		return strconv.FormatUint(uint64(vt), 10)
	case uint8:
		return strconv.FormatUint(uint64(vt), 10)
	case uint16:
		return strconv.FormatUint(uint64(vt), 10)
	case uint32:
		return strconv.FormatUint(uint64(vt), 10)
	case uint64:
		return strconv.FormatUint(vt, 10)
	case []byte:
		return string(vt)
	}

	return fmt.Sprint(val.Interface())
}

func ValidatePtr(v *reflect.Value) error {
	// sequence is very important, IsNil must be called after checking Kind() with reflect.Ptr,
	// panic otherwise
	if !v.IsValid() || v.Kind() != reflect.Ptr || v.IsNil() {
		return fmt.Errorf("not a valid pointer: %v", v)
	}

	return nil
}

func doParseKeyAndOptions(field reflect.StructField, value string) (string, *fieldOptions, error) {
	segments := strings.Split(value, ",")
	key := strings.TrimSpace(segments[0])
	options := segments[1:]

	if len(options) > 0 {
		var fieldOpts fieldOptions

		for _, segment := range options {
			option := strings.TrimSpace(segment)
			switch {
			case option == stringOption:
				fieldOpts.FromString = true
			case strings.HasPrefix(option, optionalOption):
				segs := strings.Split(option, equalToken)
				switch len(segs) {
				case 1:
					fieldOpts.Optional = true
				case 2:
					fieldOpts.Optional = true
					fieldOpts.OptionalDep = segs[1]
				default:
					return "", nil, fmt.Errorf("field %s has wrong optional", field.Name)
				}
			case option == optionalOption:
				fieldOpts.Optional = true
			case strings.HasPrefix(option, optionsOption):
				segs := strings.Split(option, equalToken)
				if len(segs) != 2 {
					return "", nil, fmt.Errorf("field %s has wrong options", field.Name)
				} else {
					fieldOpts.Options = strings.Split(segs[1], optionSeparator)
				}
			case strings.HasPrefix(option, defaultOption):
				segs := strings.Split(option, equalToken)
				if len(segs) != 2 {
					return "", nil, fmt.Errorf("field %s has wrong default option", field.Name)
				} else {
					fieldOpts.Default = strings.TrimSpace(segs[1])
				}
			case strings.HasPrefix(option, rangeOption):
				segs := strings.Split(option, equalToken)
				if len(segs) != 2 {
					return "", nil, fmt.Errorf("field %s has wrong range", field.Name)
				}
				if nr, err := parseNumberRange(segs[1]); err != nil {
					return "", nil, err
				} else {
					fieldOpts.Range = nr
				}
			}
		}

		return key, &fieldOpts, nil
	}

	return key, nil, nil
}

func maybeNewValue(field reflect.StructField, value reflect.Value) {
	if field.Type.Kind() == reflect.Ptr && value.IsNil() {
		value.Set(reflect.New(value.Type().Elem()))
	}
}

// don't modify returned fieldOptions, it's cached and shared among different calls.
func parseKeyAndOptions(tagName string, field reflect.StructField) (string, *fieldOptions, error) {
	value := field.Tag.Get(tagName)
	if len(value) == 0 {
		return field.Name, nil, nil
	} else {
		cacheLock.RLock()
		cache, ok := optionsCache[value]
		cacheLock.RUnlock()

		if ok {
			return stringx.TakeOne(cache.key, field.Name), cache.options, cache.err
		} else {
			key, options, err := doParseKeyAndOptions(field, value)
			cacheLock.Lock()
			optionsCache[value] = &optionsCacheValue{
				key:     key,
				options: options,
				err:     err,
			}
			cacheLock.Unlock()
			return stringx.TakeOne(key, field.Name), options, err
		}
	}
}

// support below notations:
// [:5] (:5] [:5) (:5)
// [1:] [1:) (1:] (1:)
// [1:5] [1:5) (1:5] (1:5)
func parseNumberRange(str string) (*numberRange, error) {
	if len(str) == 0 {
		return nil, errNumberRange
	}

	var leftInclude bool
	switch str[0] {
	case '[':
		leftInclude = true
	case '(':
		leftInclude = false
	default:
		return nil, errNumberRange
	}

	str = string(str[1:])
	if len(str) == 0 {
		return nil, errNumberRange
	}

	var rightInclude bool
	switch str[len(str)-1] {
	case ']':
		rightInclude = true
	case ')':
		rightInclude = false
	default:
		return nil, errNumberRange
	}

	str = string(str[:len(str)-1])
	fields := strings.Split(str, ":")
	if len(fields) != 2 {
		return nil, errNumberRange
	}

	if len(fields[0]) == 0 && len(fields[1]) == 0 {
		return nil, errNumberRange
	}

	var left float64
	if len(fields[0]) > 0 {
		var err error
		if left, err = strconv.ParseFloat(fields[0], 64); err != nil {
			return nil, err
		}
	} else {
		left = -math.MaxFloat64
	}

	var right float64
	if len(fields[1]) > 0 {
		var err error
		if right, err = strconv.ParseFloat(fields[1], 64); err != nil {
			return nil, err
		}
	} else {
		right = math.MaxFloat64
	}

	return &numberRange{
		left:         left,
		leftInclude:  leftInclude,
		right:        right,
		rightInclude: rightInclude,
	}, nil
}

func setValue(kind reflect.Kind, value reflect.Value, str string) error {
	if !value.CanSet() {
		return errValueNotSettable
	}

	switch kind {
	case reflect.Bool:
		value.SetBool(str == "1" || strings.ToLower(str) == "true")
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if intValue, err := strconv.ParseInt(str, 10, 64); err != nil {
			return fmt.Errorf("the value %q cannot parsed as int", str)
		} else {
			value.SetInt(intValue)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if uintValue, err := strconv.ParseUint(str, 10, 64); err != nil {
			return fmt.Errorf("the value %q cannot parsed as uint", str)
		} else {
			value.SetUint(uintValue)
		}
	case reflect.Float32, reflect.Float64:
		if floatValue, err := strconv.ParseFloat(str, 64); err != nil {
			return fmt.Errorf("the value %q cannot parsed as float", str)
		} else {
			value.SetFloat(floatValue)
		}
	case reflect.String:
		value.SetString(str)
	default:
		return errUnsupportedType
	}

	return nil
}

func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case int:
		return float64(val), true
	case int8:
		return float64(val), true
	case int16:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	case uint:
		return float64(val), true
	case uint8:
		return float64(val), true
	case uint16:
		return float64(val), true
	case uint32:
		return float64(val), true
	case uint64:
		return float64(val), true
	case float32:
		return float64(val), true
	case float64:
		return val, true
	default:
		return 0, false
	}
}

func usingDifferentKeys(field reflect.StructField, key string) bool {
	if len(field.Tag) > 0 {
		if _, ok := field.Tag.Lookup(key); !ok {
			return true
		}
	}

	return false
}
