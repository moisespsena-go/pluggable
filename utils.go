package pluggable

import (
	"reflect"

	path_helpers "github.com/moisespsena-go/path-helpers"
)

func UID(v interface{}) string {
	t := reflect.ValueOf(v).Type()
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
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

func IsOptionsProvider(v interface{}) bool {
	if plugin, ok := v.(*Plugin); ok {
		v = plugin.Value
	}
	switch v.(type) {
	case OptionProvider, OptionProviderE:
		return true
	default:
		return false
	}
}

func IsInitializador(v interface{}) bool {
	if plugin, ok := v.(*Plugin); ok {
		v = plugin.Value
	}
	switch v.(type) {
	case PluginInit, PluginInitE, PluginInitOptions, PluginInitOptionsE:
		return true
	default:
		return false
	}
}

func Filter(f func(p *Plugin) bool, plugin ...*Plugin) (result []*Plugin) {
	for _, p := range plugin {
		if f(p) {
			result = append(result, p)
		}
	}
	return
}

func Dispatcher(options *Options) PluginEventDispatcherInterface {
	return options.GetInterface(PKG + ".dispatcher").(PluginEventDispatcherInterface)
}

var Dis = Dispatcher
