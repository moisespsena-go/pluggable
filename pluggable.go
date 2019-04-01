package pluggable

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/go-errors/errors"
	"github.com/moisespsena-go/default-logger"
	"github.com/moisespsena/go-edis"
	"github.com/moisespsena/go-error-wrap"
	"github.com/moisespsena-go/path-helpers"
	"github.com/moisespsena/go-topsort"
	"github.com/op/go-logging"
)

const (
	E_REGISTER     = "register"
	E_INIT         = "init"
	E_INIT_PLUGINS = "initPlugins"
	E_INIT_DONE    = "initDone"
	E_POST_INIT    = "postInit"
)

var eof = errors.New("!eof")

type Plugin struct {
	uid                   string
	Index                 int
	Path                  string
	AbsPath               string
	Value                 interface{}
	ReflectedValue        reflect.Value
	AssetsRoot, NameSpace string
	logger                *logging.Logger
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

func (p *Plugin) Logger() *logging.Logger {
	if p.logger == nil {
		p.mu.Lock()
		defer p.mu.Unlock()
		if p.logger == nil {
			p.logger = logging.MustGetLogger(p.UID())
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
	ByUID             map[string]*Plugin
	Extensions        []Extension
	initialized       bool
	Log               *logging.Logger
	plugins           []*Plugin
	prioritaryPlugins []*Plugin
	sorted            bool
	optionsProvider   map[string]*Plugin
	befores           map[string][]string
	afters            map[string][]string
}

func NewPlugins() *Plugins {
	p := &Plugins{}
	p.SetDispatcher(p)
	p.SetOptions(NewOptions())
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

func (pls *Plugins) AddPrioritary(plugin ...interface{}) (err error) {
	if pls.sorted {
		var items []*Plugin
		err = pls.AddTo(&items, plugin...)
		pls.plugins = append(items, pls.plugins...)
		return err
	}
	return pls.AddTo(&pls.prioritaryPlugins, plugin...)
}

func (pls *Plugins) AddTo(to *[]*Plugin, plugin ...interface{}) (err error) {
	var (
		pi                interface{}
		rvalue            reflect.Value
		pth, absPath, uid string
		p                 *Plugin
		ok                bool
	)
	if pls.ByUID == nil {
		pls.ByUID = make(map[string]*Plugin)
	}

	for _, pi = range plugin {
		if piDis, ok := pi.(EventDispatcherInterface); ok {
			if piDis.Dispatcher() == nil {
				piDis.SetDispatcher(piDis)
			}
		}

		pth = path_helpers.PkgPathOf(pi)
		absPath = path_helpers.ResolveGoSrcPath(pth)

		p = &Plugin{
			Index:          len(pls.plugins),
			Path:           pth,
			AbsPath:        absPath,
			Value:          pi,
			ReflectedValue: rvalue,
		}

		uid = p.UID()
		if _, ok = pls.ByUID[uid]; ok {
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
			pls.ByUID[p.UID()] = p

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
			case PluginRegisterDisArg:
				r.OnRegister(pls.PluginDispatcher())
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

func (pls *Plugins) sort() (err error) {
	var (
		graph      = topsort.NewGraph()
		provider   = map[string]string{}
		pluginsMap = map[string]*Plugin{}
		afters     = pls.afters
		befors     = pls.befores
		uidOrPanic = func(v interface{}) string {
			var uid string

			switch vt := v.(type) {
			case string:
				uid = vt
			default:
				uid = UID(vt)
			}

			if _, ok := pluginsMap[uid]; !ok {
				panic(fmt.Errorf("Plugin %q not registered", uid))
			}
			return uid
		}
	)
	pls.sorted = true
	log.Debug("sort")

	if afters == nil {
		afters = map[string][]string{}
	}
	if befors == nil {
		befors = map[string][]string{}
	}

	for _, p := range pls.plugins {
		uid := p.UID()
		graph.AddNode(uid)
		pluginsMap[uid] = p
		if provides, ok := p.Value.(PluginProvideOptions); ok {
			for _, optionName := range provides.ProvideOptions() {
				// if have previous provider, order it
				if prevId, ok := provider[optionName]; ok {
					graph.AddEdge(uid, prevId)
				}
				provider[optionName] = uid
			}
		}
	}

	globalOptions := pls.options

	for _, p := range pls.plugins {
		if requires, ok := p.Value.(PluginRequireOptions); ok {
			err = pls.doPlugin(p, func(p *Plugin) error {
				uid := p.UID()
				for _, optionName := range requires.RequireOptions() {
					if _, ok := globalOptions.Get(optionName); !ok {
						providedBy, ok := provider[optionName]
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

		if after, ok := afters[p.UID()]; ok {
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

		if before, ok := befors[p.UID()]; ok {
			for _, v := range before {
				graph.AddEdge(uidOrPanic(v), p.UID())
			}
		}
	}

	result, err := graph.TopSort()
	if err != nil {
		return errwrap.Wrap(err, "Top-Sort")
	}

	plugins := make([]*Plugin, len(result))
	for i, uid := range result {
		plugins[i] = pluginsMap[uid]
	}

	pls.plugins = append(pls.prioritaryPlugins, plugins...)
	pls.optionsProvider = make(map[string]*Plugin)

	for optionName, uid := range provider {
		pls.optionsProvider[optionName] = pluginsMap[uid]
	}

	log.Debug("sort done")

	return nil
}

func (pls *Plugins) Sort() (err error) {
	if !pls.sorted {
		return pls.sort()
	}
	return SortedError
}

func (pls *Plugins) initPlugin(p *Plugin) (err error) {
	log.Debug("init plugin", p.String())
	options := pls.Options()

	if requireOptions, ok := p.Value.(PluginRequireOptions); ok {
		for _, name := range requireOptions.RequireOptions() {
			if !options.Has(name) {
				return fmt.Errorf("Required option %q, provided by %s is <nil>", name, pls.optionsProvider[name])
			}
		}
	}

	if l, ok := p.Value.(LogSetter); ok {
		l.SetLog(defaultlogger.NewLogger(p.UID()))
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
	case PluginInitEDis:
		pl.Init(pls)
	case PluginInitEDisE:
		err = pl.Init(pls)
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

	err = pls.Sort()
	if err != nil && err != SortedError {
		return
	}
	err = nil

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

	err = pls.Each(pls.initPlugin)

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
