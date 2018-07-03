package pluggable

import (
	"github.com/op/go-logging"
	"github.com/qor/qor"
)

type PluginRegister interface {
	OnRegister(dis PluginEventDispatcherInterface)
}

type PluginInit interface {
	Init() error
}

type PluginInitOptions interface {
	Init(options *Options) error
}

type PluginInitPlugins interface {
	Init(plugins *Plugins) error
}

type PluginInitSite interface {
	InitSite(site qor.SiteInterface) error
}

type PluginProvideOptions interface {
	ProvideOptions() []string
}

type PluginRequireOptions interface {
	RequireOptions() []string
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
