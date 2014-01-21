package types

import (
	"fmt"
	"reflect"
)

// ToString transforms the given value into
// a string. If the value is a nil pointer
// or a nil interface, it returns the empty
// string.
func ToString(val interface{}) string {
	if val == nil {
		return ""
	}
	switch x := val.(type) {
	case string:
		return x
	case fmt.Stringer:
		return x.String()
	case error:
		return x.Error()
	}
	v := reflect.ValueOf(val)
	if k := v.Type().Kind(); k == reflect.Ptr || k == reflect.Interface {
		if v.IsNil() {
			return ""
		}
	}
	return fmt.Sprintf("%v", val)
}

// ToInt tries to convert its argument to an integer. It will convert
// bool, uint and its varities, floats and even strings if it can parse
// them.
func ToInt(val interface{}) (int, error) {
	iv := reflect.ValueOf(val)
	if !iv.IsValid() {
		return 0, fmt.Errorf("invalid value")
	}
	v := reflect.Indirect(iv)
	switch v.Kind() {
	case reflect.String:
	case reflect.Bool:
		if v.Bool() {
			return 1, nil
		}
		return 0, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int(v.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int(v.Uint()), nil
	case reflect.Float32, reflect.Float64:
		return int(v.Float()), nil
	}
	return 0, fmt.Errorf("can't convert %v to int", v.Type())
}

// Equal is a shortcut for reflect.DeepEqual.
func Equal(obj1, obj2 interface{}) bool {
	return reflect.DeepEqual(obj1, obj2)
}

// IsTrue returns whether the value is 'true', in the sense of not the zero of its type,
// and whether the value has a meaningful truth value. This function is a copy of the
// one found in text/template
func IsTrue(value interface{}) (truth, ok bool) {
	return IsTrueVal(reflect.ValueOf(value))
}

// IsTrueVal works like IsTrue, but accepts a reflect.Value rather
// than an interface{}.
func IsTrueVal(val reflect.Value) (truth, ok bool) {
	if !val.IsValid() {
		// Something like var x interface{}, never set. It's a form of nil.
		return false, true
	}
	switch val.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		truth = val.Len() > 0
	case reflect.Bool:
		truth = val.Bool()
	case reflect.Complex64, reflect.Complex128:
		truth = val.Complex() != 0
	case reflect.Chan, reflect.Func, reflect.Ptr, reflect.Interface:
		truth = !val.IsNil()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		truth = val.Int() != 0
	case reflect.Float32, reflect.Float64:
		truth = val.Float() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		truth = val.Uint() != 0
	case reflect.Struct:
		truth = true // Struct values are always true.
	default:
		return
	}
	return truth, true
}

// IsInt returns true iff typ is any of the int types.
func IsInt(typ reflect.Type) bool {
	k := typ.Kind()
	return k == reflect.Int || k == reflect.Int8 || k == reflect.Int16 || k == reflect.Int32 || k == reflect.Int64
}

// IsUint returns true iff typ is any of the uint types.
func IsUint(typ reflect.Type) bool {
	k := typ.Kind()
	return k == reflect.Uint || k == reflect.Uint8 || k == reflect.Uint16 || k == reflect.Uint32 || k == reflect.Uint64
}

// IsFloat returns true iff typ is any of the float types.
func IsFloat(typ reflect.Type) bool {
	k := typ.Kind()
	return k == reflect.Float32 || k == reflect.Float64
}
