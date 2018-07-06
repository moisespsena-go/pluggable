package pluggable

import (
	"fmt"
	"reflect"
)

func PUID(v interface{}) string {
	t := reflect.ValueOf(v).Type().Elem()
	id := fmt.Sprintf("%v.%v", t.PkgPath(), t.Name())
	if named, ok := v.(NamedPlugin); ok {
		id += "#" + named.Name()
	}
	return id
}
