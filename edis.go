package pluggable

import (
	"fmt"

	"github.com/moisespsena/go-edis"
	"github.com/moisespsena/go-error-wrap"
)

type EventDispatcher = edis.EventDispatcher
type Event = edis.Event

type PluginEventInterface interface {
	edis.EventInterface
	Plugins() PluginEventDispatcherInterface
	Plugin() *Plugin
}

type PluginEvent struct {
	edis.EventInterface
	plugins PluginEventDispatcherInterface
	plugin  *Plugin
}

func (pe *PluginEvent) Plugins() PluginEventDispatcherInterface {
	return pe.plugins
}

func (pe *PluginEvent) Plugin() *Plugin {
	return pe.plugin
}

type PluginEventCallbackInterface interface {
	Call(pe PluginEventInterface) error
}

type PluginCallbackFuncE func(pe PluginEventInterface) error

func (p PluginCallbackFuncE) Call(pe PluginEventInterface) error {
	return p(pe)
}

type PluginCallbackFunc func(pe PluginEventInterface)

func (p PluginCallbackFunc) Call(pe PluginEventInterface) error {
	p(pe)
	return nil
}

type PluginEventDispatcherInterface interface {
	edis.EventDispatcherInterface
	OnPlugin(eventName string, callbacks ...interface{}) error
	TriggerPlugins(e edis.EventInterface, plugins ...*Plugin) (err error)
	EachPlugins(items []*Plugin, cb func(plugin *Plugin) (err error)) (err error)
	EachPluginsCallback(items []*Plugin, callbacks ...func(plugin *Plugin) error) (err error)
}

type PluginEventDispatcher struct {
	edis.EventDispatcher
}

type PluginEventCallback func(e PluginEventInterface) error

func (c PluginEventCallback) Call(e PluginEventInterface) error {
	return c(e)
}

func (ped PluginEventDispatcher) OnPlugin(eventName string, callbacks ...interface{}) error {
	cbLocal := func(cbi PluginEventCallbackInterface) edis.CallbackFuncE {
		return edis.CallbackFuncE(func(e edis.EventInterface) error {
			return cbi.Call(e.(PluginEventInterface))
		})
	}
	var cbi PluginEventCallbackInterface
	for _, cb := range callbacks {
		switch t := cb.(type) {
		case PluginEventCallbackInterface:
			cbi = t
		case func(e PluginEventInterface) error:
			cbi = PluginCallbackFuncE(t)
		case func(e PluginEventInterface):
			cbi = PluginCallbackFunc(t)
		default:
			return fmt.Errorf("Invalid Callback type %s", t)
		}
		ped.On("plugin:"+eventName, cbLocal(cbi))
	}
	return nil
}

func (ped *PluginEventDispatcher) TriggerPlugins(e edis.EventInterface, plugins ...*Plugin) (err error) {
	pe := &PluginEvent{e, ped, nil}
	eLocal := &PluginEvent{&edis.Event{PName: "plugin:" + e.Name(), PParent: pe}, ped, nil}
	err = ped.EachPluginsCallback(plugins, func(plugin *Plugin) (err error) {
		if ed, ok := plugin.Value.(edis.EventDispatcherInterface); ok {
			pe.plugin = plugin
			eLocal.plugin = plugin
			ped.Trigger(eLocal)
			if err = ed.Trigger(pe); err == nil {
				if e.Error() != nil {
					return e.Error()
				}
			}
		}
		return nil
	})
	if err != nil && e.Error() == nil {
		err = errwrap.Wrap(err, "Trigger %v", e.Name())
		e.SetError(err)
	}
	return
}

func (ped *PluginEventDispatcher) EachPluginsCallback(items []*Plugin, callbacks ...func(plugin *Plugin) error) (err error) {
	err = ped.EachPlugins(items, func(plugin *Plugin) (err error) {
		for _, cb := range callbacks {
			err = cb(plugin)
			if err != nil {
				return errwrap.Wrap(err, "Callback %s", cb)
			}
			if err != nil {
				return
			}
		}
		return
	})
	return
}

func (ped *PluginEventDispatcher) EachPlugins(items []*Plugin, cb func(plugin *Plugin) (err error)) (err error) {
	for _, plugin := range items {
		err = cb(plugin)
		if err != nil {
			return errwrap.Wrap(err, "Plugin %s", plugin)
		}
	}
	return
}
