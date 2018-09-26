package pluggable

import (
	"path"

	"github.com/moisespsena/go-assetfs/api"
	"github.com/moisespsena/go-error-wrap"
	"github.com/moisespsena/go-path-helpers"
)

var E_LOCALE_FS = PREFIX + ".localeFS"

type I18nPluginsInterface interface {
	PluginFSInterface
	LocaleFS() api.Interface
}

type I18nPlugins struct {
	PluginsFS
	localeFS api.Interface
}

func (pls *I18nPlugins) LocaleFS() api.Interface {
	return pls.localeFS
}

func InitPluginI18nFS(pls I18nPluginsInterface) {
	pls.OnPlugin("register", func(e PluginEventInterface) error {
		p := e.Plugin()
		fs := pls.Dispatcher().(I18nPluginsInterface).LocaleFS()
		if p.AbsPath != "" {
			pth := path.Join(p.AssetsRoot, "locale")
			if path_helpers.IsExistingDir(pth) {
				fs.NameSpace(p.NameSpace).RegisterPath(pth)
			}
		}

		if dis, ok := p.Value.(EventDispatcherInterface); ok {
			e := &LocaleFSEvent{NewPluginEvent(E_LOCALE_FS), fs.NameSpace(p.NameSpace)}
			e.SetPlugin(p)
			if err := dis.Trigger(e); err != nil {
				return errwrap.Wrap(err, "Trigger ", E_LOCALE_FS)
			}
		}
		return nil
	})
}

func NewI18nPlugins(fs api.Interface) *I18nPlugins {
	pls := &I18nPlugins{*NewPluginsFS(fs), fs.NameSpace("locale")}
	pls.SetDispatcher(pls)
	InitPluginI18nFS(pls)
	return pls
}

type LocaleFSEvent struct {
	PluginEventInterface
	LocaleFS api.Interface
}

func (e *LocaleFSEvent) RegisterWithNameSpace(nameSpace, basePath string) error {
	pth := path.Join(basePath, "locale")
	if path_helpers.IsExistingDir(pth) {
		return e.LocaleFS.NameSpace(nameSpace).RegisterPath(pth)
	}
	return nil
}

func OnLocaleFS(p EventDispatcherInterface, cb func(e *LocaleFSEvent)) {
	p.On(E_LOCALE_FS, func(e PluginEventInterface) {
		cb(e.(*LocaleFSEvent))
	})
}
