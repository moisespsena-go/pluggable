package pluggable

import (
	"reflect"

	"github.com/moisespsena-go/path-helpers"
)

func UID(v interface{}) string {
	t := reflect.ValueOf(v).Type().Elem()
	id := path_helpers.PkgPathOf(t)
	if named, ok := v.(NamedPlugin); ok {
		id += "#" + named.Name()
	} else if t.Name() != "Plugin" {
		id += "." + t.Name()
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
