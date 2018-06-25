package module

import (
	"path"

	"github.com/moisespsena/go-assetfs/api"
	"github.com/moisespsena/go-path-helpers"
)

type I18nPlugins struct {
	*Plugins
	LocaleFS api.Interface
}

func NewI18nPlugins(assetFS api.Interface) *I18nPlugins {
	pls := &I18nPlugins{NewPlugins(assetFS), assetFS.NameSpace("locale")}
	pls.OnPluginRegisterCallback(pls.onPluginRegister)
	return pls
}

func (pls *I18nPlugins) onPluginRegister(plugin *Plugin) error {
	if plugin.AbsPath != "" {
		pth := path.Join(plugin.AbsPath, "assets", "locale")
		if path_helpers.IsExistingDir(pth) {
			pls.LocaleFS.NameSpace(plugin.Path).RegisterPath(pth)
		}
	}
	return nil
}
