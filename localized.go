package pluggable

import (
	"path"

	"github.com/moisespsena/go-assetfs/api"
	"github.com/moisespsena/go-path-helpers"
)

type I18nPlugins struct {
	PluginsFS
	LocaleFS api.Interface
}

func NewI18nPlugins(fs api.Interface) *I18nPlugins {
	pls := &I18nPlugins{*NewPluginsFS(fs), fs.NameSpace("locale")}
	pls.OnPlugin("register", func(e PluginEventInterface) error {
		plugin := e.Plugin()
		if plugin.AbsPath != "" {
			pth := path.Join(plugin.AbsPath, "assets", "locale")
			if path_helpers.IsExistingDir(pth) {
				pls.LocaleFS.NameSpace(plugin.Path).RegisterPath(pth)
			}
		}
		return nil
	})
	return pls
}
