package terminal

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type Flag interface {
	Parse(string) (name string, val any, err error)
}

type FlagBase[T any] struct {
	Name  string
	Alias []string
	value T
}

func (f FlagBase[T]) Parse(s string) (name string, val any, err error) {
	var zero T
	tp := reflect.TypeOf(zero)
	value := reflect.New(tp).Elem()
	switch tp {
	case reflect.TypeOf(uint64(0)):
		var val uint64
		var err error
		if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
			s = s[2:]
			val, err = strconv.ParseUint(s, 16, 64)
		} else {
			val, err = strconv.ParseUint(s, 10, 64)
		}

		if err != nil {
			return "", nil, fmt.Errorf("error parsing %s as uint64: %v", s, err)
		}

		value.SetUint(val)
	case reflect.TypeOf(string("")):
		val := s
		value.SetString(val)
	case reflect.TypeOf(bool(false)):
		val, _ := strconv.ParseBool(s)
		value.SetBool(val)
	default:
		panic("unsupported type")
	}
	f.value = value.Interface().(T)
	return f.Name, f.value, nil
}
