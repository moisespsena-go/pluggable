package pluggable

import (
	"github.com/moisespsena-go/logging"
)

type PluginRegister interface {
	OnRegister()
}

type PluginRegisterArg interface {
	OnRegister(p *Plugin)
}

type PluginRegisterOptionsArg interface {
	OnRegister(options *Options)
}

type PluginInit interface {
	Init()
}

type PluginInitE interface {
	Init() error
}

type PluginInitOptions interface {
	Init(options *Options)
}


type PluginInitOptionsE interface {
	Init(options *Options) error
}

type PluginProvideOptions interface {
	ProvideOptions() []string
}

type PluginRequireOptions interface {
	RequireOptions() []string
}

type PluginBeforeUID interface {
	Before() []string
}

type PluginBeforeI interface {
	Before() []interface{}
}

type PluginAfterUID interface {
	After() []string
}

type PluginAfterI interface {
	After() []interface{}
}

type NamedPlugin interface {
	Name() string
}

type LogSetter interface {
	SetLog(log logging.Logger)
}

type LoggedInterface interface {
	LoggerSetter
	Log() logging.Logger
}

type GlobalOptionsInterface interface {
	SetGlobalOptions(options *Options)
	GetGlobalOptions() *Options
}

type PluginFSNameSpace interface {
	NameSpace() string
}

type PluginAssetsRootPath interface {
	AssetsRootPath() string
}

type PluginSetter interface {
	SetPlugin(p *Plugin)
}

type PluginAccess interface {
	PluginSetter
	Plugin() *Plugin
}

type LoggerSetter interface {
	SetLogger(Log logging.Logger)
}

type OptionProvider interface {
	ProvidesOptions(options *Options)
}

type OptionProviderE interface {
	ProvidesOptions(options *Options) (err error)
}
