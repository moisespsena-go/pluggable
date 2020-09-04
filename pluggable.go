package pluggable

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/go-errors/errors"
	defaultlogger "github.com/moisespsena-go/default-logger"
	"github.com/moisespsena-go/edis"
	errwrap "github.com/moisespsena-go/error-wrap"
	path_helpers "github.com/moisespsena-go/path-helpers"

	"github.com/moisespsena-go/logging"
)

const (
	E_REGISTER     = "register"
	E_INIT         = "init"
	E_INIT_PLUGINS = "initPlugins"
	E_INIT_DONE    = "initDone"
	E_POST_INIT    = "postInit"
)

var eof = errors.New("!eof")

type PluginsMap map[string]*Plugin

func (this *PluginsMap) Add(plugin ...*Plugin) {
	if *this == nil {
		*this = map[string]*Plugin{}
	}
	for _, p := range plugin {
		(*this)[p.UID()] = p
	}
}

func (this *PluginsMap) Get(uid string) *Plugin {
	if *this == nil {
		return nil
	}
	return (*this)[uid]
}

func (this *PluginsMap) Has(uid string) bool {
	if *this == nil {
		return false
	}
	if _, ok := (*this)[uid]; ok {
		return true
	}
	return false
}

type Plugin struct {
	uid                   string
	Index                 int
	Path                  string
	AbsPath               string
	Value                 interface{}
	ReflectedValue        reflect.Value
	AssetsRoot, NameSpace string
	logger                logging.Logger
	mu                    sync.Mutex
}

func (p *Plugin) UID() string {
	if p.uid == "" {
		p.uid = UID(p.Value)
	}
	return p.uid
}

func (p *Plugin) String() string {
	return p.UID()
}

func (p *Plugin) Logger() logging.Logger {
	if p.logger == nil {
		p.mu.Lock()
		defer p.mu.Unlock()
		if p.logger == nil {
			p.logger = logging.GetOrCreateLogger(p.UID())
		}
	}
	return p.logger
}

func (p *Plugin) SetLoggerLevel(level logging.Level) {
	logging.SetLevel(level, p.UID())
}

type Plugins struct {
	Logged
	PluginEventDispatcher
	ByUID           PluginsMap
	Extensions      []Extension
	initialized     bool
	plugins         []*Plugin
	sorted          bool
	optionsProvider map[string]*Plugin
	befores         map[string][]string
	afters          map[string][]string
}

func NewPlugins() *Plugins {
	p := &Plugins{}
	p.PluginEventDispatcher.PluginsGetter = func() []*Plugin {
		return p.plugins
	}
	p.SetDispatcher(p)
	p.SetOptions(NewOptions())
	if p.log == nil {
		p.log = log
	}
	return p
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
	log := logging.WithPrefix(logging.WithPrefix(pls.log, "trigger"), e.Name())
	log.Debug("start")
	defer log.Debug("done")
	if len(plugins) == 0 {
		if len(pls.plugins) > 0 {
			return pls.PluginEventDispatcher.TriggerPlugins(e, pls.plugins...)
		}
		return nil
	}
	return pls.PluginEventDispatcher.TriggerPlugins(e, plugins...)
}

func (pls *Plugins) Each(cb func(plugin *Plugin) (err error)) (err error) {
	if !pls.initialized {
		return fmt.Errorf("Plugins has not be initialized")
	}
	return pls.EachPlugins(pls.plugins, cb)
}

func (pls *Plugins) Add(plugin ...interface{}) (err error) {
	return pls.AddTo(&pls.plugins, plugin...)
}

func (pls *Plugins) AddTo(to *[]*Plugin, plugin ...interface{}) (err error) {
	var (
		pi                interface{}
		rvalue            reflect.Value
		pth, absPath, uid string
		p                 *Plugin
	)

	for _, pi = range plugin {
		if piDis, ok := pi.(EventDispatcherInterface); ok {
			if piDis.Dispatcher() == nil {
				piDis.SetDispatcher(piDis)
			}
		}

		pth = path_helpers.PkgPathOf(pi)
		_, absPath = path_helpers.ResolveGoSrcPath(pth)

		p = &Plugin{
			Index:          len(pls.plugins),
			Path:           pth,
			AbsPath:        absPath,
			Value:          pi,
			ReflectedValue: rvalue,
		}

		uid = p.UID()
		if pls.ByUID.Has(uid) {
			log.Warningf("%q Duplicated. Ignored.", uid)
			continue
		}

		if pi, ok := pi.(PluginAccess); ok {
			pi.SetPlugin(p)
		}

		err = func() (err error) {
			defer func() {
				if err != nil {
					err = errwrap.Wrap(err, "Plugin %q", p.UID())
				}
			}()
			*to = append(*to, p)
			pls.ByUID.Add(p)

			if setter, ok := pi.(PluginSetter); ok {
				setter.SetPlugin(p)
			}

			if setter, ok := pi.(LoggerSetter); ok {
				setter.SetLogger(p.Logger())
			}

			switch r := pi.(type) {
			case PluginRegister:
				r.OnRegister()
			case PluginRegisterArg:
				r.OnRegister(p)
			case PluginRegisterOptionsArg:
				r.OnRegister(pls.options)
			}

			err = pls.TriggerPlugins(edis.NewEvent(E_REGISTER), p)

			if pls.initialized {
				err = pls.initPlugin(p)
			}
			return
		}()
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

func (pls *Plugins) After(self, other interface{}) {
	if pls.afters == nil {
		pls.afters = map[string][]string{}
	}
	var selfUID, otherUID string

	if s, ok := self.(string); ok {
		selfUID = s
	} else {
		selfUID = UID(self)
	}
	if s, ok := other.(string); ok {
		otherUID = s
	} else {
		otherUID = UID(other)
	}

	if _, ok := pls.afters[selfUID]; !ok {
		pls.afters[selfUID] = []string{otherUID}
	} else {
		pls.afters[selfUID] = append(pls.afters[selfUID], otherUID)
	}
}

func (pls *Plugins) Before(self, other interface{}) {
	if pls.befores == nil {
		pls.befores = map[string][]string{}
	}
	var selfUID, otherUID string

	if s, ok := self.(string); ok {
		selfUID = s
	} else {
		selfUID = UID(self)
	}
	if s, ok := other.(string); ok {
		otherUID = s
	} else {
		otherUID = UID(other)
	}

	if _, ok := pls.befores[selfUID]; !ok {
		pls.befores[selfUID] = []string{otherUID}
	} else {
		pls.befores[selfUID] = append(pls.befores[selfUID], otherUID)
	}
}

func (pls *Plugins) sortProviders() (providers []*Plugin, err error) {
	var (
		provider = map[string]string{}
	)
	log.Debug("sort for provides")
	defer log.Debug("sort for provides done")
	var sorter = &Sorter{
		PluginsMap: pls.ByUID,
		Plugins:    Filter(func(p *Plugin) bool { return IsOptionsProvider(p) }, pls.plugins...),
		Afters:     pls.afters,
		Befores:    pls.befores,
		Pre: func(state *SorterState) error {
			for _, p := range state.Plugins {
				uid := p.UID()
				state.Graph.AddNode(uid)
				state.pluginsMap[uid] = p
				if provides, ok := p.Value.(PluginProvideOptions); ok {
					for _, optionName := range provides.ProvideOptions() {
						// if have previous provider, order it
						if prevId, ok := provider[optionName]; ok {
							state.Graph.AddEdge(uid, prevId)
						}
						provider[optionName] = uid
					}
				}
			}
			return nil
		},
		Post: func(state *SorterState) error {
			for optionName, uid := range provider {
				pls.optionsProvider[optionName] = state.pluginsMap[uid]
			}
			return nil
		},
	}
	providers, err = sorter.Sort(func(state *SorterState, p *Plugin) (err error) {
		if requires, ok := p.Value.(PluginRequireOptions); ok {
			err = pls.doPlugin(p, func(p *Plugin) error {
				uid := p.UID()
				for _, optionName := range requires.RequireOptions() {
					if optionName == "" {
						panic(fmt.Errorf("empty option name from %s (%T)", uid, p.Value))
					}
					if _, ok := pls.options.Get(optionName); !ok {
						providedBy, ok := provider[optionName]
						if !ok {
							return fmt.Errorf("Option %q, required by %s, does not have provedor.", optionName, p)
						}
						state.Graph.AddEdge(uid, providedBy)
					}
				}
				return nil
			})
			if err != nil {
				return
			}
		}
		return
	})
	return
}

func (pls *Plugins) ProvideOptions() (err error) {
	log.Debug("provides")
	defer log.Debug("provides done")
	var providers []*Plugin
	if providers, err = pls.sortProviders(); err != nil {
		return
	}
	for _, p := range providers {
		switch provider := p.Value.(type) {
		case OptionProvider:
			provider.ProvidesOptions(pls.options)
		case OptionProviderE:
			if err = provider.ProvidesOptions(pls.options); err != nil {
				return errwrap.Wrap(err, "Plugin {%v} Provides failed", p.String())
			}
		}
	}
	return
}

func (pls *Plugins) sortf(state *SorterState, p *Plugin) (err error) {
	graph, uidOrPanic := state.Graph, state.UidOrPanic
	if after, ok := p.Value.(PluginAfterUID); ok {
		for _, v := range after.After() {
			graph.AddEdge(p.UID(), uidOrPanic(v))
		}
	}

	if after, ok := p.Value.(PluginAfterI); ok {
		for _, v := range after.After() {
			graph.AddEdge(p.UID(), uidOrPanic(v))
		}
	}

	if after, ok := state.Afters[p.UID()]; ok {
		for _, v := range after {
			graph.AddEdge(p.UID(), uidOrPanic(v))
		}
	}

	if before, ok := p.Value.(PluginBeforeUID); ok {
		for _, v := range before.Before() {
			graph.AddEdge(uidOrPanic(v), p.UID())
		}
	}

	if before, ok := p.Value.(PluginBeforeI); ok {
		for _, v := range before.Before() {
			graph.AddEdge(uidOrPanic(v), p.UID())
		}
	}

	if before, ok := state.Befors[p.UID()]; ok {
		for _, v := range before {
			graph.AddEdge(uidOrPanic(v), p.UID())
		}
	}
	return
}

func (pls *Plugins) sortForInit() (sorted []*Plugin, err error) {
	pls.log.Debug("sort for init")
	defer log.Debug("sort for init done")
	var sorter = &Sorter{
		PluginsMap: pls.ByUID,
		Plugins: pls.plugins,
		Afters:  pls.afters,
		Befores: pls.befores,
	}
	sorted, err = sorter.Sort(pls.sortf)
	return
}

func (pls *Plugins) initPlugin(p *Plugin) (err error) {
	log := logging.WithPrefix(logging.WithPrefix(pls.log, "init plugin"), p.String())
	log.Debug("start")
	defer log.Debug("done")
	options := pls.Options()

	if requireOptions, ok := p.Value.(PluginRequireOptions); ok {
		for _, name := range requireOptions.RequireOptions() {
			if name == "" {
				return fmt.Errorf("required option key from %s (%T) is blank", p.uid, p.Value)
			}
			if !options.Has(name) {
				return fmt.Errorf("Required option %q, provided by %s is <nil>", name, pls.optionsProvider[name])
			}
		}
	}

	if l, ok := p.Value.(LogSetter); ok {
		l.SetLog(defaultlogger.GetOrCreateLogger(p.UID()))
	}

	if gOptions, ok := p.Value.(GlobalOptionsInterface); ok {
		gOptions.SetGlobalOptions(options)
	}
	err = pls.TriggerPlugins(edis.NewEvent(E_INIT), p)
	if err != nil {
		return err
	}

	switch pl := p.Value.(type) {
	case PluginInit:
		pl.Init()
	case PluginInitE:
		err = pl.Init()
		if err != nil {
			return errwrap.Wrap(err, "Init")
		}
	case PluginInitOptions:
		pl.Init(options)
	case PluginInitOptionsE:
		err = pl.Init(options)
		if err != nil {
			return errwrap.Wrap(err, "Init")
		}
	}

	if err = pls.TriggerPlugins(edis.NewEvent("init"), p); err != nil {
		return errwrap.Wrap(err, "Init")
	}
	err = pls.TriggerPlugins(edis.NewEvent(E_INIT_DONE), p)
	return
}

func (pls *Plugins) Init() (err error) {
	if pls.initialized {
		return Initialized
	}
	pls.initialized = true

	var sorted []*Plugin

	if sorted, err = pls.sortForInit(); err != nil {
		return
	}

	pls.plugins = sorted

	log.Debug("init extensions")

	for _, extension := range pls.Extensions {
		err = extension.Init(pls)
		if err != nil {
			return errwrap.Wrap(err, "Init Extencion %T", extension)
		}
	}

	log.Debug("init extensions done")

	err = pls.Trigger(edis.NewEvent("init"))
	if err != nil {
		return
	}

	log.Debug("init plugins")

	for _, p := range sorted {
		if IsInitializador(p) {
			if err = pls.initPlugin(p); err != nil {
				return
			}
		}
	}

	log.Debug("init plugins done")

	if err != nil {
		return
	}

	err = pls.Trigger(edis.NewEvent(E_INIT_DONE))
	if err != nil {
		return errwrap.Wrap(err, "Plugins > Init > Trigger:initDone")
	}
	pls.TriggerPlugins(edis.NewEvent(E_POST_INIT))
	return errwrap.Wrap(err, "Plugins > Init > Trigger:postInit")
}