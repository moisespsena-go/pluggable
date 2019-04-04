package pluggable

import (
	"path"

	"github.com/moisespsena-go/assetfs/assetfsapi"
	"github.com/moisespsena-go/error-wrap"
	"github.com/moisespsena-go/path-helpers"
)

var E_LOCALE_FS = PKG + ".localeFS"

type I18nPluginsInterface interface {
	PluginFSInterface
	LocaleFS() assetfsapi.Interface
}

type I18nPlugins struct {
	PluginsFS
	localeFS assetfsapi.Interface
}

func (pls *I18nPlugins) LocaleFS() assetfsapi.Interface {
	return pls.localeFS
}

func InitPluginI18nFS(pls I18nPluginsInterface) {
	pls.OnPlugin("register", func(e PluginEventInterface) error {
		p := e.Plugin()
		fs := pls.Dispatcher().(I18nPluginsInterface).LocaleFS()
		if registrator, ok := fs.(assetfsapi.PathRegistrator); ok && p.AbsPath != "" {
			pth := path.Join(p.AssetsRoot, "locale")
			if path_helpers.IsExistingDir(pth) {
				registrator.NameSpace(p.NameSpace).(assetfsapi.PathRegistrator).RegisterPath(pth)
			}
		}

		if dis, ok := p.Value.(EventDispatcherInterface); ok {
			e := &LocaleFSEvent{NewPluginEvent(E_LOCALE_FS), fs}
			e.SetPlugin(p)
			if err := dis.Trigger(e); err != nil {
				return errwrap.Wrap(err, "Trigger ", E_LOCALE_FS)
			}
		}
		return nil
	})
}

func NewI18nPlugins(fs assetfsapi.Interface) *I18nPlugins {
	pls := &I18nPlugins{*NewPluginsFS(fs), fs.NameSpace("@locale")}
	pls.SetDispatcher(pls)
	InitPluginI18nFS(pls)
	return pls
}

type LocaleFSEvent struct {
	PluginEventInterface
	LocaleFS assetfsapi.Interface
}

func (e *LocaleFSEvent) RegisterWithNameSpace(nameSpace, basePath string) error {
	pth := path.Join(basePath, "locale")
	if path_helpers.IsExistingDir(pth) {
		return e.LocaleFS.NameSpace(nameSpace).(assetfsapi.PathRegistrator).RegisterPath(pth)
	}
	return nil
}

func OnLocaleFS(p EventDispatcherInterface, cb func(e *LocaleFSEvent)) {
	p.On(E_LOCALE_FS, func(e PluginEventInterface) {
		cb(e.(*LocaleFSEvent))
	})
}
