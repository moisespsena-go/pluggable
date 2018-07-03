package pluggable

import (
	"path"

	"github.com/moisespsena/go-assetfs/api"
)

type PluginsFS struct {
	Plugins
	AssetFS api.Interface
}

func NewPluginsFS(fs api.Interface) *PluginsFS {
	pls := &PluginsFS{AssetFS: fs}
	pls.OnPlugin("register", func(e PluginEventInterface) error {
		if e.Plugin().AbsPath != "" {
			pls.AssetFS.RegisterPath(path.Join(e.Plugin().AbsPath, "assets"))
		}
		return nil
	})
	return pls
}
