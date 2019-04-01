package pluggable

import (
	"path"

	"github.com/moisespsena-go/error-wrap"

	"reflect"

	"github.com/moisespsena/go-assetfs/assetfsapi"
	"github.com/moisespsena-go/path-helpers"
)

var E_FS = PKG + ".FS"

type PluginFSInterface interface {
	PluginEventDispatcherInterface
	SetAssetFSPathRegister(regiser func(fs assetfsapi.PathRegistrator, pth string) error)
	AssetFSPathRegister() func(fs assetfsapi.PathRegistrator, pth string) error
	FS() assetfsapi.Interface
	PrivateFS() assetfsapi.Interface
	PluginPrivateFS(pluginID string) assetfsapi.Interface
}

type PluginsFS struct {
	*Plugins
	fs               assetfsapi.Interface
	pathRegisterFunc func(fs assetfsapi.PathRegistrator, pth string) error
}

func (p *PluginsFS) SetAssetFSPathRegister(register func(fs assetfsapi.PathRegistrator, pth string) error) {
	p.pathRegisterFunc = register
}

func (p *PluginsFS) AssetFSPathRegister() func(fs assetfsapi.PathRegistrator, pth string) error {
	return p.pathRegisterFunc
}

func (p *PluginsFS) FS() assetfsapi.Interface {
	return p.fs
}

func (p *PluginsFS) PrivateFS() assetfsapi.Interface {
	return p.fs.NameSpace("@private")
}

func (p *PluginsFS) PluginPrivateFS(pluginUID string) assetfsapi.Interface {
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
			if registrator, ok := pls.FS().(assetfsapi.PathRegistrator); ok {
				register(registrator, path.Join(p.AssetsRoot, "assets"))
			}
			if registrator, ok := pfs.(assetfsapi.PathRegistrator); ok {
				register(registrator, path.Join(p.AssetsRoot, "data"))
			}
		}

		if dis, ok := p.Value.(EventDispatcherInterface); ok {
			e := &FSEvent{NewPluginEvent(E_FS), pls.FS(), register, pfs}
			e.SetPlugin(p)
			if err := dis.Trigger(e); err != nil {
				err = errwrap.Wrap(err, "Trigger ", E_FS)
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

func NewPluginsFS(fs assetfsapi.Interface) *PluginsFS {
	pls := &PluginsFS{Plugins: NewPlugins(), fs: fs}
	pls.SetDispatcher(pls)
	pls.pathRegisterFunc = DefaultFSPathRegister
	InitPluginFS(pls)
	return pls
}

func DefaultFSPathRegister(fs assetfsapi.PathRegistrator, pth string) error {
	return fs.RegisterPath(pth)
}

type PluginPrivateFSSetter interface {
	SetPrivateFS(fs assetfsapi.Interface)
}

type PluginAssetFSSetter interface {
	SetPrivateFS(fs assetfsapi.Interface)
}

type FSEvent struct {
	PluginEventInterface
	AssetFS       assetfsapi.Interface
	AssetRegister func(fs assetfsapi.PathRegistrator, pth string) error
	PrivateFS     assetfsapi.Interface
}

func (FSEvent) PathOf(value interface{}) (pth string) {
	t := reflect.TypeOf(value)
	for t.Kind() != reflect.Struct {
		t = t.Elem()
	}
	pth = t.PkgPath()
	_, pth = path_helpers.ResolveGoSrcPath(pth)
	return
}

func (a *FSEvent) RegisterAssetPath(basePath string) {
	if r, ok := a.AssetFS.(assetfsapi.PathRegistrator); ok {
		a.AssetRegister(r, path.Join(basePath, "assets"))
	}
}

func OnFS(p EventDispatcherInterface, cb func(e *FSEvent)) {
	p.On(E_FS, func(e PluginEventInterface) {
		cb(e.(*FSEvent))
	})
}

type FSSetter interface {
	SetFS(fs assetfsapi.Interface)
}
