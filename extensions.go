package module

import (
	"github.com/qor/qor"
)

type Extension interface {
	Init(plugins *Plugins) error
	OnPluginRegister(plugin *Plugin) error
	OnPluginInit(plugin *Plugin) error
	OnPluginInitSite(plugin *Plugin, site qor.SiteInterface) error
}
