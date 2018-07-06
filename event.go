package pluggable

import "github.com/moisespsena/go-edis"

type PluginEventInterface interface {
	EventInterface
	PluginDispatcher() PluginEventDispatcherInterface
	SetPluginDispatcher(dis PluginEventDispatcherInterface)
	Plugin() *Plugin
	SetPlugin(*Plugin)
	Options() *Options
	SetOptions(*Options)
}

type PluginEvent struct {
	EventInterface
	pluginDispatcher PluginEventDispatcherInterface
	plugin           *Plugin
	options          *Options
}

type Parent struct {
	Value EventInterface
}

func NewPluginEvent(e interface{}, data ...interface{}) (pe *PluginEvent) {
	switch et := e.(type) {
	case string:
		pe = &PluginEvent{EventInterface: edis.NewEvent(et, data...)}
	default:
		pe = &PluginEvent{EventInterface: e.(EventInterface)}
	}
	for _, d := range data {
		switch dt := d.(type) {
		case Parent:
			pe.SetParent(dt.Value)
		default:
			pe.SetData(d)
		}
	}
	return
}

func (pe *PluginEvent) SetPluginDispatcher(dis PluginEventDispatcherInterface) {
	pe.pluginDispatcher = dis
}

func (pe *PluginEvent) PluginDispatcher() PluginEventDispatcherInterface {
	return pe.pluginDispatcher
}

func (pe *PluginEvent) Plugin() *Plugin {
	return pe.plugin
}

func (pe *PluginEvent) SetPlugin(p *Plugin) {
	pe.plugin = p
}

func (pe *PluginEvent) Options() *Options {
	return pe.options
}

func (pe *PluginEvent) SetOptions(o *Options) {
	pe.options = o
}
