package pluggable

import (
	"path"

	"reflect"

	"github.com/moisespsena/go-assetfs/api"
	"github.com/moisespsena/go-error-wrap"
	"github.com/moisespsena/go-path-helpers"
)

const E_ASSET_FS = "pluggable.AssetFS"

type PluginFSInterface interface {
	EventDispatcherInterface
	SetAssetFSPathRegister(regiser func(fs api.Interface, pth string) error)
	AssetFSPathRegister() func(fs api.Interface, pth string) error
}

type PluginsFS struct {
	*Plugins
	AssetFS          api.Interface
	pathRegisterFunc func(fs api.Interface, pth string) error
}

func (p *PluginsFS) SetAssetFSPathRegister(register func(fs api.Interface, pth string) error) {
	p.pathRegisterFunc = register
}

func (p *PluginsFS) AssetFSPathRegister() func(fs api.Interface, pth string) error {
	return p.pathRegisterFunc
}

func NewPluginsFS(fs api.Interface) *PluginsFS {
	pls := &PluginsFS{Plugins: NewPlugins(), AssetFS: fs}
	pls.SetDispatcher(pls)
	pls.pathRegisterFunc = DefaultFSPathRegister
	pls.OnPlugin("register", func(e PluginEventInterface) error {
		register := pls.Dispatcher().(PluginFSInterface).AssetFSPathRegister()
		p := e.Plugin()
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
			register(pls.AssetFS, path.Join(p.AssetsRoot, "assets"))
		}

		if dis, ok := p.Value.(EventDispatcherInterface); ok {
			e := &AssetFSEvent{NewPluginEvent(E_ASSET_FS), fs, register}
			e.SetPlugin(p)
			if err := dis.Trigger(e); err != nil {
				return errwrap.Wrap(err, "Trigger ", E_ASSET_FS)
			}
		}
		return nil
	})
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
