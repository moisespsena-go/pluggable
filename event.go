package pluggable

import "github.com/moisespsena/go-edis"

type PluginEventInterface interface {
	EventInterface
	PluginDispatcher() PluginEventDispatcherInterface
	Plugin() *Plugin
	SetPlugin(*Plugin)
	Options() *Options
	SetOptions(*Options)
	WithPluginDispatcher(dis PluginEventDispatcherInterface) func()
}

type PluginEvent struct {
	EventInterface
	plugin     *Plugin
	options    *Options
	dispatcher PluginEventDispatcherInterface
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
		pe.SetData(d)
		break
	}
	return
}

func (pe *PluginEvent) PluginDispatcher() PluginEventDispatcherInterface {
	return pe.dispatcher
}

func (pe *PluginEvent) Plugin() *Plugin {
	return pe.plugin
}

func (pe *PluginEvent) SetPlugin(p *Plugin) {
	pe.plugin = p
}

func (pe *PluginEvent) Options() *Options {
	if pe.options == nil {
		return pe.dispatcher.Options()
	}
	return pe.options
}

func (pe *PluginEvent) SetOptions(o *Options) {
	pe.options = o
}

func (pe *PluginEvent) WithPluginDispatcher(dis PluginEventDispatcherInterface) func() {
	old := pe.dispatcher
	pe.dispatcher = dis
	return func() {
		pe.dispatcher = old
	}
}
