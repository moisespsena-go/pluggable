package pluggable

import (
	"path"

	"github.com/moisespsena/go-error-wrap"

	"reflect"

	"github.com/moisespsena/go-assetfs/api"
	"github.com/moisespsena/go-path-helpers"
)

var E_ASSET_FS = PREFIX + ".AssetFS"

type PluginFSInterface interface {
	PluginEventDispatcherInterface
	SetAssetFSPathRegister(regiser func(fs api.Interface, pth string) error)
	AssetFSPathRegister() func(fs api.Interface, pth string) error
	FS() api.Interface
	PrivateFS() api.Interface
	PluginPrivateFS(pluginID string) api.Interface
}

type PluginsFS struct {
	*Plugins
	fs               api.Interface
	pathRegisterFunc func(fs api.Interface, pth string) error
}

func (p *PluginsFS) SetAssetFSPathRegister(register func(fs api.Interface, pth string) error) {
	p.pathRegisterFunc = register
}

func (p *PluginsFS) AssetFSPathRegister() func(fs api.Interface, pth string) error {
	return p.pathRegisterFunc
}

func (p *PluginsFS) FS() api.Interface {
	return p.fs
}

func (p *PluginsFS) PrivateFS() api.Interface {
	return p.fs.NameSpace("@private")
}

func (p *PluginsFS) PluginPrivateFS(pluginUID string) api.Interface {
	fs := p.PrivateFS()
	fs = fs.NameSpace(p.ByUID[pluginUID].Path)
	return fs
}

func InitPluginFS(pls PluginFSInterface) {
	pls.OnPlugin("register", func(e PluginEventInterface) (err error) {
		register := pls.Dispatcher().(PluginFSInterface).AssetFSPathRegister()
		p := e.Plugin()
		pfs := pls.PluginPrivateFS(p.UID())

		if p.AbsPath != "" {
			if assetsPath, ok := p.Value.(PluginAssetsRootPath); ok {
				p.AssetsRoot = assetsPath.AssetsRootPath()
			} else {
				p.AssetsRoot = e.Plugin().AbsPath
			}

			p.NameSpace = p.Path
			if ns, ok := p.Value.(PluginFSNameSpace); ok {
				p.NameSpace = ns.NameSpace()
			}
			register(pls.FS(), path.Join(p.AssetsRoot, "assets"))
			register(pfs, path.Join(p.AssetsRoot, "data"))
		}

		if dis, ok := p.Value.(EventDispatcherInterface); ok {
			e := &AssetFSEvent{NewPluginEvent(E_ASSET_FS), pls.FS(), register}
			e.SetPlugin(p)
			if err := dis.Trigger(e); err != nil {
				err = errwrap.Wrap(err, "Trigger ", E_ASSET_FS)
			}
		}

		if err == nil {
			if setter, ok := p.Value.(FSSetter); ok {
				setter.SetFS(pfs)
			}
		}

		return nil
	})
}

func NewPluginsFS(fs api.Interface) *PluginsFS {
	pls := &PluginsFS{Plugins: NewPlugins(), fs: fs}
	pls.SetDispatcher(pls)
	pls.pathRegisterFunc = DefaultFSPathRegister
	InitPluginFS(pls)
	return pls
}

func DefaultFSPathRegister(fs api.Interface, pth string) error {
	return fs.RegisterPath(pth)
}

type AssetFSEvent struct {
	PluginEventInterface
	AssetFS  api.Interface
	Register func(fs api.Interface, pth string) error
}

func (AssetFSEvent) PathOf(value interface{}) (pth string) {
	t := reflect.TypeOf(value)
	for t.Kind() != reflect.Struct {
		t = t.Elem()
	}
	pth = t.PkgPath()
	pth = path_helpers.ResolveGoSrcPath(pth)
	return
}

func (a *AssetFSEvent) RegisterAssets(basePath string) {
	a.Register(a.AssetFS, path.Join(basePath, "assets"))
}

func OnAssetFS(p EventDispatcherInterface, cb func(e *AssetFSEvent)) {
	p.On(E_ASSET_FS, func(e PluginEventInterface) {
		cb(e.(*AssetFSEvent))
	})
}

type FSSetter interface {
	SetFS(fs api.Interface)
}
