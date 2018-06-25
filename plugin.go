package module

import (
	"fmt"
	"path"
	"reflect"

	"github.com/moisespsena/go-assetfs"
	"github.com/moisespsena/go-error-wrap"
	"github.com/moisespsena/go-topsort"
	"github.com/op/go-logging"
	"github.com/qor/helpers"
	"github.com/qor/qor"
)

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
	AssetFS                   assetfs.Interface
	Items                     []*Plugin
	ByPath                    map[string]*Plugin
	Extensions                []Extension
	initialized               bool
	AfterPluginsInitCallbacks []func() error
	AfterPluginInitCallbacks  []func(plugin *Plugin) error
	OnPluginRegisterCallbacks []func(plugin *Plugin) error
	OnPluginSetupCallbacks    []func(plugin *Plugin) error
	OnPluginInitCallbacks     []func(plugin *Plugin) error
	Log                       *logging.Logger
}

func NewPlugins(assetFS assetfs.Interface) *Plugins {
	return &Plugins{AssetFS: assetFS}
}

func (pls *Plugins) Extension(extensions ...Extension) (err error) {
	pls.Extensions = append(pls.Extensions, extensions...)
	if pls.initialized {
		for _, extension := range extensions {
			extension.Init(pls)
			err = pls.EachCallback(extension.OnPluginRegister)
			if err != nil {
				return errwrap.Wrap(err, "Extensio %v", extension)
			}
		}
	}
	return
}

func (pls *Plugins) AfterPluginsInitCallback(callbacks ...func() error) {
	pls.AfterPluginsInitCallbacks = append(pls.AfterPluginsInitCallbacks, callbacks...)
}

func (pls *Plugins) AfterPluginInitCallback(callbacks ...func(plugin *Plugin) error) {
	pls.AfterPluginInitCallbacks = append(pls.AfterPluginInitCallbacks, callbacks...)
}

func (pls *Plugins) OnPluginRegisterCallback(callbacks ...func(plugin *Plugin) error) (err error) {
	pls.OnPluginRegisterCallbacks = append(pls.OnPluginRegisterCallbacks, callbacks...)
	return errwrap.Wrap(pls.EachCallback(callbacks...), "RegisterCallback")
}

func (pls *Plugins) OnPluginInitCallback(callbacks ...func(plugin *Plugin) error) (err error) {
	pls.OnPluginInitCallbacks = append(pls.OnPluginInitCallbacks, callbacks...)
	if pls.initialized {
		return errwrap.Wrap(pls.EachCallback(callbacks...), "InitCallback")
	}
	return
}

func (pls *Plugins) OnPluginSetupCallback(callbacks ...func(plugin *Plugin) error) (err error) {
	pls.OnPluginSetupCallbacks = append(pls.OnPluginSetupCallbacks, callbacks...)
	if pls.initialized {
		return errwrap.Wrap(pls.EachCallback(callbacks...), "SetupCallback")
	}
	return nil
}

func (pls *Plugins) EachCallback(callbacks ...func(plugin *Plugin) error) (err error) {
	err = pls.Each(func(plugin *Plugin) (err error) {
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

func (pls *Plugins) Each(cb func(plugin *Plugin) (err error)) (err error) {
	for _, plugin := range pls.Items {
		err = cb(plugin)
		if err != nil {
			return errwrap.Wrap(err, "Plugin %s", plugin)
		}
	}
	return
}

func (pls *Plugins) Add(plugin ...interface{}) (err error) {
	if pls.ByPath == nil {
		pls.ByPath = make(map[string]*Plugin)
	}

	for _, pi := range plugin {
		rvalue := reflect.Indirect(reflect.ValueOf(pi))
		pth := rvalue.Type().PkgPath()
		var absPath string
		if absPath = helpers.ResolveGoSrcPath(pth); absPath != "" {
			pls.AssetFS.RegisterPath(path.Join(absPath, "assets"))
		}
		p := &Plugin{"", len(pls.Items), pth, absPath, pi, rvalue}
		pls.Items = append(pls.Items, p)
		pls.ByPath[pth] = p

		if onRegister, ok := pi.(PluginRegister); ok {
			onRegister.OnRegister(pls, p)
		}

		for _, extension := range pls.Extensions {
			err = extension.OnPluginRegister(p)
			if err != nil {
				return
			}
		}

		for _, cb := range pls.OnPluginRegisterCallbacks {
			err = cb(p)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (pls *Plugins) Setup() (err error) {
	return pls.Each(func(p *Plugin) (err error) {
		if setup, ok := p.Value.(Setup); ok {
			err = setup.Setup()
			if err != nil {
				return errwrap.Wrap(err, "Setup")
			}
		}
		return
	})
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

	for _, p := range pls.Items {
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

	for _, p := range pls.Items {
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

	pls.Items = plugins

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

	pls.Each(func(p *Plugin) (err error) {
		log.Debug("Init plugin", p.String())
		if gOptions, ok := p.Value.(GlobalOptionsInterface); ok {
			gOptions.SetGlobalOptions(globalOptions)
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
		for _, extension := range pls.Extensions {
			err = extension.OnPluginInit(p)
			if err != nil {
				return errwrap.Wrap(err, "Extension %T > OnPluginInit %s", extension, p)
			}
		}
		for _, cb := range pls.OnPluginInitCallbacks {
			err = cb(p)
			if err != nil {
				return errwrap.Wrap(err, "PluginInitCallback %T > Call %s", cb, p)
			}
		}
		return
	})

	for _, cb := range pls.AfterPluginsInitCallbacks {
		err = cb()
		if err != nil {
			return errwrap.Wrap(err, "After Plugins Init Callback %s", cb)
		}
	}

	for _, p := range pls.Items {
		for _, cb := range pls.AfterPluginInitCallbacks {
			err = cb(p)
			if err != nil {
				return errwrap.Wrap(err, "After Plugin Init Callback %s > Plugin %s", cb, p)
			}
		}
	}

	return errwrap.Wrap(err, "Plugins > Init")
}
