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

type PluginAfter interface {
	After() []string
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
