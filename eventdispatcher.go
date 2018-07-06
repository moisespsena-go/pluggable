package pluggable

import (
	"fmt"

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
	options   *Options
	dispacher PluginEventDispatcherInterface
}

func (ped *PluginEventDispatcher) SetDispatcher(dispatcher PluginEventDispatcherInterface) {
	ped.dispacher = dispatcher
}

func (ped *PluginEventDispatcher) Dispatcher() PluginEventDispatcherInterface {
	return ped.dispacher
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

func (ped *PluginEventDispatcher) OnPluginE(eventName string, callbacks ...interface{}) error {
	cbLocal := func(cbi PluginEventCallbackInterface) CallbackFuncE {
		return CallbackFuncE(func(e EventInterface) error {
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

func (ped *PluginEventDispatcher) TriggerPlugins(e EventInterface, plugins ...*Plugin) (err error) {
	dispatcher := ped.dispacher
	if dispatcher == nil {
		dispatcher = ped
	}
	var (
		pe PluginEventInterface
		ok bool
	)
	if pe, ok = e.(PluginEventInterface); !ok || pe.PluginDispatcher() != nil {
		pe = &PluginEvent{EventInterface: e}
	}

	pe.SetDispatcher(dispatcher)
	pe.SetPluginDispatcher(dispatcher)
	pe.SetOptions(dispatcher.Options())

	eLocal := &PluginEvent{&Event{PName: "plugin:" + e.Name(), PParent: pe}, dispatcher, nil, dispatcher.Options()}
	err = ped.EachPluginsCallback(plugins, func(plugin *Plugin) (err error) {
		if ed, ok := plugin.Value.(EventDispatcherInterface); ok {
			pe.SetPlugin(plugin)
			eLocal.plugin = plugin
			if err = ped.Trigger(eLocal); err == nil {
				if eLocal.Error() != nil {
					return eLocal.Error()
				}
			}
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
