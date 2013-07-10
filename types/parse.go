package types

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
)

// Parse tries to parse an string into the given argument.
// e.g.
//     var f float32
//     Parse("27.5", &f)
//     var width uint
//     Parse("57", &width)
// Supported types are: bool, u?int(8|16|32|64)? and float(32|64). If
// the parsed value would overflow the given type, the maximum value
// (or minimum, if it's negative) for the type will be set.
func Parse(val string, arg interface{}) error {
	// If val is empty, do nothing
	if val == "" {
		return nil
	}
	v := reflect.ValueOf(arg)
	for v.Type().Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if !v.CanSet() {
		return fmt.Errorf("Invalid argument type passed to Parse(). Please pass %s instead of %s.",
			reflect.PtrTo(v.Type()), v.Type())

	}
	switch v.Type().Kind() {
	case reflect.Bool:
		res := false
		if val != "" && val != "0" && strings.ToLower(val) != "false" {
			res = true
		}
		v.SetBool(res)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		res, err := strconv.ParseInt(val, 0, 64)
		if err != nil {
			return err
		}
		if v.OverflowInt(res) {
			if res > 0 {
				res = int64(math.Pow(2, float64(8*v.Type().Size()-1)) - 1)
			} else {
				res = -int64(math.Pow(2, float64(8*v.Type().Size()-1)))
			}
		}
		v.SetInt(res)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		res, err := strconv.ParseUint(val, 0, 64)
		if err != nil {
			return err
		}
		if v.OverflowUint(res) {
			res = uint64(math.Pow(2, float64(8*v.Type().Size())) - 1)
		}
		v.SetUint(res)
	case reflect.Float32, reflect.Float64:
		res, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return err
		}
		v.SetFloat(res)
	case reflect.String:
		v.SetString(val)
	default:
		return fmt.Errorf("Invalid argument type passed to Parse(): %s. Please, see the documentation for a list of the supported types.",
			v.Type())
	}
	return nil
}