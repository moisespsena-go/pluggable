package pluggable

import (
	"path"

	"github.com/moisespsena/go-assetfs/api"
	"github.com/moisespsena/go-path-helpers"
)

type I18nPluginsInterface interface {
	LocaleFS() api.Interface
}

type I18nPlugins struct {
	PluginsFS
	localeFS api.Interface
}

func (pls *I18nPlugins) LocaleFS() api.Interface {
	return pls.localeFS
}

func NewI18nPlugins(fs api.Interface) *I18nPlugins {
	pls := &I18nPlugins{*NewPluginsFS(fs), fs.NameSpace("locale")}
	pls.SetDispatcher(pls)
	pls.OnPlugin("register", func(e PluginEventInterface) error {
		plugin := e.Plugin()
		if plugin.AbsPath != "" {
			pth := path.Join(plugin.AssetsRoot, "locale")
			if path_helpers.IsExistingDir(pth) {
				pls.Dispatcher().(I18nPluginsInterface).LocaleFS().NameSpace(plugin.NameSpace).RegisterPath(pth)
			}
		}
		return nil
	})
	return pls
}
