package pluggable

import (
	"fmt"
	"reflect"

	"github.com/moisespsena/go-edis"
	"github.com/moisespsena/go-error-wrap"
)

type EventDispatcher struct {
	edis.EventDispatcher
}

func (ed *EventDispatcher) OnE(eventName string, callbacks ...interface{}) error {
	return ed.EventDispatcher.OnE(eventName, prepareCallbacks(callbacks)...)
}

func (ed *EventDispatcher) On(eventName string, callbacks ...interface{}) {
	ed.EventDispatcher.On(eventName, prepareCallbacks(callbacks)...)
}

func prepareCallbacks(callbacks []interface{}) []interface{} {
	for i, cb := range callbacks {
		switch cbt := cb.(type) {
		case func(e PluginEventInterface):
			callbacks[i] = PluginEventCallback(cbt)
		case func(e PluginEventInterface) error:
			callbacks[i] = PluginEventCallbackE(cbt)
		}
	}
	return callbacks
}

type PluginEventDispatcherInterface interface {
	EventDispatcherInterface
	OnPluginE(eventName string, callbacks ...interface{}) error
	OnPlugin(eventName string, callbacks ...interface{})
	TriggerPlugins(e EventInterface, plugins ...*Plugin) (err error)
	EachPlugins(items []*Plugin, cb func(plugin *Plugin) (err error)) (err error)
	EachPluginsCallback(items []*Plugin, callbacks ...func(plugin *Plugin) error) (err error)
	Options() *Options
}

type PluginEventDispatcher struct {
	EventDispatcher
	options *Options
}

func (ped *PluginEventDispatcher) SetOptions(options *Options) {
	ped.options = options
}

func (ped *PluginEventDispatcher) Options() *Options {
	return ped.options
}

func (ped *PluginEventDispatcher) OnPlugin(eventName string, callbacks ...interface{}) {
	if err := ped.OnPluginE(eventName, callbacks...); err != nil {
		panic(err)
	}
}

func (ped *PluginEventDispatcher) OnPluginE(eventName string, callbacks ...interface{}) (err error) {
	cbLocal := func(cbi PluginEventCallbackInterface) edis.CallbackFuncE {
		return edis.CallbackFuncE(func(e EventInterface) error {
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
		err = ped.OnE("plugin:"+eventName, cbLocal(cbi))
		if err != nil {
			return
		}
	}
	return nil
}

func (ped *PluginEventDispatcher) PluginDispatcher() PluginEventDispatcherInterface {
	return ped.Dispatcher().(PluginEventDispatcherInterface)
}

func (ped *PluginEventDispatcher) TriggerPlugins(e EventInterface, plugins ...*Plugin) (err error) {
	var (
		pe PluginEventInterface
		ok bool
	)
	if pe, ok = e.(PluginEventInterface); !ok || pe.PluginDispatcher() != nil {
		pe = &PluginEvent{EventInterface: e}
	}

	dis := ped.PluginDispatcher()

	if pe.PluginDispatcher() == nil {
		defer pe.WithPluginDispatcher(dis)()
	}

	eLocal := &PluginEvent{&Event{PName: "plugin:" + e.Name()}, nil, dis.Options(), dis}
	err = ped.EachPluginsCallback(plugins, func(plugin *Plugin) (err error) {
		eLocal.plugin = plugin
		if err = ped.Trigger(eLocal); err == nil {
			if eLocal.Error() != nil {
				return eLocal.Error()
			}
		}
		if ed, ok := plugin.Value.(EventDispatcherInterface); ok {
			pe.SetPlugin(plugin)
			if err != nil {
				return
			}
			if err = ed.Trigger(pe); err == nil {
				if pe.Error() != nil {
					return pe.Error()
				}
			}
			if err != nil {
				return
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
				typ := reflect.TypeOf(cb)
				return errwrap.Wrap(err, "Callback %s [%T]", typ.Name(), cb)
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
