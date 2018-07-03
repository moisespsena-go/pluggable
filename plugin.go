package pluggable

import (
	"fmt"
	"reflect"

	"github.com/go-errors/errors"
	"github.com/moisespsena/go-edis"
	"github.com/moisespsena/go-error-wrap"
	"github.com/moisespsena/go-topsort"
	"github.com/op/go-logging"
	"github.com/qor/helpers"
)

const (
	E_PLUGIN_TRIGGER  = "pluginTrigger"
	E_PLUGIN_REGISTER = "register"
	E_PLUGIN_INIT     = "init"
)

var eof = errors.New("!eof")

type Plugin struct {
	uid            string
	Index          int
	Path           string
	AbsPath        string
	Value          interface{}
	ReflectedValue reflect.Value
}

func (p *Plugin) UID() string {
	if p.uid == "" {
		t := p.ReflectedValue.Type()
		p.uid = fmt.Sprintf("%v.%v", t.PkgPath(), t.Name())
		if named, ok := p.Value.(NamedPlugin); ok {
			p.uid += "#" + named.Name()
		}
	}
	return p.uid
}

func (p *Plugin) String() string {
	return p.UID()
}

type Plugins struct {
	GlobalOptions
	PluginEventDispatcher
	ByPath      map[string]*Plugin
	Extensions  []Extension
	initialized bool
	Log         *logging.Logger
	plugins []*Plugin
}

func NewPlugins() *Plugins {
	return &Plugins{}
}

func (pls *Plugins) Extension(extensions ...Extension) (err error) {
	pls.Extensions = append(pls.Extensions, extensions...)
	if pls.initialized {
		for _, extension := range extensions {
			extension.Init(pls)
			if ed, ok := extension.(edis.EventDispatcherInterface); ok {
				err = pls.Each(func(plugin *Plugin) (err error) {
					return ed.Trigger(edis.NewEvent("pluginRegister", plugin))
				})
				if err != nil {
					return errwrap.Wrap(err, "Extension %v", extension)
				}
			}
		}
	}
	return
}

func (pls *Plugins) TriggerPlugins(e edis.EventInterface, plugins ...*Plugin) (err error) {
	if len(plugins) == 0 {
		if len(pls.plugins) > 0 {
			return pls.PluginEventDispatcher.TriggerPlugins(e, pls.plugins...)
		}
		return nil
	}
	return pls.PluginEventDispatcher.TriggerPlugins(e, plugins...)
}

func (pls *Plugins) Each(cb func(plugin *Plugin) (err error)) (err error) {
	return pls.EachPlugins(pls.plugins, cb)
}

func (pls *Plugins) Add(plugin ...interface{}) (err error) {
	if pls.ByPath == nil {
		pls.ByPath = make(map[string]*Plugin)
	}

	for _, pi := range plugin {
		rvalue := reflect.Indirect(reflect.ValueOf(pi))
		pth := rvalue.Type().PkgPath()
		var absPath string
		absPath = helpers.ResolveGoSrcPath(pth)
		p := &Plugin{"", len(pls.plugins), pth, absPath, pi, rvalue}
		pls.plugins = append(pls.plugins, p)
		pls.ByPath[pth] = p

		if r, ok := pi.(PluginRegister); ok {
			r.OnRegister(pls)
		}

		err = pls.TriggerPlugins(edis.NewEvent(E_PLUGIN_REGISTER), p)
	}
	return nil
}

func (pls *Plugins) doPlugin(p *Plugin, f func(p *Plugin) (err error)) (err error) {
	err = f(p)
	if err != nil {
		err = errwrap.Wrap(err, "Plugin {%v}", p.String())
	}
	return
}

func (pls *Plugins) sort() (err error) {
	graph := topsort.NewGraph()
	provideMap := map[string]string{}
	byUID := map[string]*Plugin{}

	for _, p := range pls.plugins {
		uid := p.UID()
		graph.AddNode(uid)
		byUID[uid] = p
		if provides, ok := p.Value.(PluginProvideOptions); ok {
			for _, optionName := range provides.ProvideOptions() {
				// if have previous provider, order it
				if prevId, ok := provideMap[optionName]; ok {
					graph.AddEdge(uid, prevId)
				}
				provideMap[optionName] = uid
			}
		}
	}

	globalOptions := pls.GlobalOptions.GlobalOptions

	for _, p := range pls.plugins {
		if requires, ok := p.Value.(PluginRequireOptions); ok {
			err = pls.doPlugin(p, func(p *Plugin) error {
				uid := p.UID()
				for _, optionName := range requires.RequireOptions() {
					if _, ok := globalOptions.Get(optionName); !ok {
						providedBy, ok := provideMap[optionName]
						if !ok {
							return fmt.Errorf("Option %q, required by %s, does not have provedor.", optionName, p)
						}
						graph.AddEdge(uid, providedBy)
					}
				}
				return nil
			})
			if err != nil {
				return
			}
		}
	}

	result, err := graph.TopSort()
	if err != nil {
		return errwrap.Wrap(err, "Top-Sort")
	}

	plugins := make([]*Plugin, len(result))
	for i, uid := range result {
		plugins[i] = byUID[uid]
	}

	pls.plugins = plugins

	return nil
}

func (pls *Plugins) Init() (err error) {
	if pls.initialized {
		return nil
	}
	pls.initialized = true

	for _, extension := range pls.Extensions {
		err = extension.Init(pls)
		if err != nil {
			return errwrap.Wrap(err, "Init Extencion %T", extension)
		}
	}

	log.Debug("Sort plugins")
	err = pls.sort()
	if err != nil {
		return errwrap.Wrap(err, "Init > sort")
	}

	globalOptions := pls.GlobalOptions.GlobalOptions

	err = pls.Trigger(edis.NewEvent("init"))
	if err != nil {
		return
	}

	err = pls.Each(func(p *Plugin) (err error) {
		log.Debug("Init plugin", p.String())
		if gOptions, ok := p.Value.(GlobalOptionsInterface); ok {
			gOptions.SetGlobalOptions(globalOptions)
		}
		err = pls.TriggerPlugins(edis.NewEvent("init"), p)
		if err != nil {
			return err
		}
		if pl, ok := p.Value.(PluginInit); ok {
			err = pl.Init()
			if err != nil {
				return errwrap.Wrap(err, "Init")
			}
		} else if pl, ok := p.Value.(PluginInitPlugins); ok {
			err = pl.Init(pls)
			if err != nil {
				return errwrap.Wrap(err, "Init")
			}
		} else if pl, ok := p.Value.(PluginInitOptions); ok {
			err = pl.Init(globalOptions)
			if err != nil {
				return errwrap.Wrap(err, "Init")
			}
		}
		err = pls.TriggerPlugins(edis.NewEvent("initDone"), p)
		return
	})

	if err != nil {
		return
	}

	err = pls.Trigger(edis.NewEvent("initDone"))
	return errwrap.Wrap(err, "Plugins > Init")
}
