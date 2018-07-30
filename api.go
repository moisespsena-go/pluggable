package pluggable

import (
	"github.com/op/go-logging"
)

type PluginRegister interface {
	OnRegister()
}

type PluginRegisterArg interface {
	OnRegister(p *Plugin)
}

type PluginRegisterDisArg interface {
	OnRegister(dis PluginEventDispatcherInterface)
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

type PluginInitEDisE interface {
	Init(dis PluginEventDispatcherInterface) error
}

type PluginInitEDis interface {
	Init(plugins PluginEventDispatcherInterface)
}

type PluginProvideOptions interface {
	ProvideOptions() []string
}

type PluginRequireOptions interface {
	RequireOptions() []string
}

type PluginBefore interface {
	Before() []string
}

type PluginBeforeI interface {
	Before() []interface{}
}

type PluginAfter interface {
	After() []string
}

type PluginAfterI interface {
	After() []interface{}
}

type NamedPlugin interface {
	Name() string
}

type LoggedInterface interface {
	Log() *logging.Logger
	SetLog(log *logging.Logger)
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
