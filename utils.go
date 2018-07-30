package pluggable

import (
	"fmt"
	"reflect"
)

func UID(v interface{}) string {
	t := reflect.ValueOf(v).Type().Elem()
	id := fmt.Sprintf("%v.%v", t.PkgPath(), t.Name())
	if named, ok := v.(NamedPlugin); ok {
		id += "#" + named.Name()
	}
	return id
}

func UIDs(vs ...interface{}) []string {
	r := make([]string, len(vs))
	for i, v := range vs {
		r[i] = UID(v)
	}
	return r
}
