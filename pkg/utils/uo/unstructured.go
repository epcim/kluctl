package uo

import (
	"fmt"
	"github.com/ohler55/ojg/jp"
	"reflect"
)

func MergeStrMap(a map[string]string, b map[string]string) {
	for k, v := range b {
		a[k] = v
	}
}

func CopyMergeStrMap(a map[string]string, b map[string]string) map[string]string {
	c := make(map[string]string)
	MergeStrMap(c, a)
	MergeStrMap(c, b)
	return c
}

func GetChild(parent interface{}, key interface{}) (interface{}, bool, error) {
	if m, ok := getDict(parent); ok {
		keyStr, ok := key.(string)
		if !ok {
			return nil, false, fmt.Errorf("key is not a string")
		}
		v, found := m[keyStr]
		return v, found, nil
	} else if m, ok := parent.(jp.Keyed); ok {
		keyStr, ok := key.(string)
		if !ok {
			return nil, false, fmt.Errorf("key is not a string")
		}
		v, found := m.ValueForKey(keyStr)
		return v, found, nil
	}

	v := reflect.ValueOf(parent)
	if v.Type().Kind() == reflect.Slice {
		keyInt, ok := key.(int)
		if !ok {
			return nil, false, fmt.Errorf("key is not an int")
		}
		if keyInt < 0 || keyInt >= v.Len() {
			return nil, false, fmt.Errorf("index out of bounds")
		}
		e := v.Index(keyInt)
		return e.Interface(), true, nil
	}
	return nil, false, fmt.Errorf("unknown parent type")
}

func SetChild(parent interface{}, key interface{}, value interface{}) error {
	if m, ok := getDict(parent); ok {
		keyStr, ok := key.(string)
		if !ok {
			return fmt.Errorf("key is not a string")
		}
		m[keyStr] = value
		return nil
	} else if m, ok := parent.(jp.Keyed); ok {
		keyStr, ok := key.(string)
		if !ok {
			return fmt.Errorf("key is not a string")
		}
		m.SetValueForKey(keyStr, value)
		return nil
	}

	v := reflect.ValueOf(parent)
	if v.Type().Kind() == reflect.Slice {
		keyInt, ok := key.(int)
		if !ok {
			return fmt.Errorf("key is not an int")
		}

		e := v.Index(keyInt)
		e.Set(reflect.ValueOf(value))
		return nil
	}
	return fmt.Errorf("unknown parent type")
}
