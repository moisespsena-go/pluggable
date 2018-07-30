package pluggable

import (
	"path"

	"github.com/moisespsena/go-assetfs/api"
)

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
			pls.Dispatcher().(PluginFSInterface).AssetFSPathRegister()(pls.AssetFS, path.Join(p.AssetsRoot, "assets"))
		}
		return nil
	})
	return pls
}

func DefaultFSPathRegister(fs api.Interface, pth string) error {
	return fs.RegisterPath(pth)
}
