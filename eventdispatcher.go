package pluggable

import (
	"fmt"
	"reflect"

	"github.com/moisespsena-go/edis"
	errwrap "github.com/moisespsena-go/error-wrap"
	"github.com/moisespsena-go/logging"
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
	GetPlugins() []*Plugin
}

type PluginEventDispatcher struct {
	EventDispatcher
	options       *Options
	PluginsGetter func() []*Plugin
}

func (ped *PluginEventDispatcher) GetPlugins() []*Plugin {
	return ped.PluginsGetter()
}

func (ped *PluginEventDispatcher) SetOptions(options *Options) {
	if ped.options != nil {
		ped.options.Del(PKG + ".dispatcher")
	}

	options.Set(PKG+".dispatcher", ped.PluginDispatcher())
	ped.options = options
}

func (ped *PluginEventDispatcher) Options() *Options {
	return ped.options
}

func (ped *PluginEventDispatcher) SetDispatcher(dis EventDispatcherInterface) {
	ped.EventDispatcher.SetDispatcher(dis)
	if ped.options != nil {
		ped.options.Set(PKG+".dispatcher", dis.(PluginEventDispatcherInterface))
	}
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
		log_ := ped.Logger()
		if log_ == nil {
			log_ = log
		}
		log_ = logging.WithPrefix(log_, "trigger -> " + e.Name())
		log_.Debug("start")
		defer log_.Debug("done")
		eLocal.plugin = plugin
		if err = func()(err error) {
			log_.Debug("local -> start")
			defer log_.Debug("local -> done")
			if err = ped.Trigger(eLocal); err == nil {
				if eLocal.Error() != nil {
					return eLocal.Error()
				}
			}
			return
		}(); err != nil {
			return
		}
		if ed, ok := plugin.Value.(EventDispatcherInterface); ok {
			pe.SetPlugin(plugin)
			if err != nil {
				return
			}
			msg := "value -> "+fmt.Sprintf("%T", plugin.Value) + " -> "
			log_.Debug(msg+"start")
			defer log_.Debug(msg+"done")
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
