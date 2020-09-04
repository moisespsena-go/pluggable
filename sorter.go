package pluggable

import (
	"fmt"

	errwrap "github.com/moisespsena-go/error-wrap"
	"github.com/moisespsena-go/topsort"
)

type SorterState struct {
	Plugins        []*Plugin
	Graph          *topsort.Graph
	pluginsMap     PluginsMap
	Befors, Afters map[string][]string
}

func (this SorterState) UidOrPanic(v interface{}) string {
	var uid string

	switch vt := v.(type) {
	case string:
		uid = vt
	default:
		uid = UID(vt)
	}

	if _, ok := this.pluginsMap[uid]; !ok {
		panic(fmt.Errorf("Plugin %q not registered", uid))
	}
	return uid
}

type Sorter struct {
	PluginsMap
	Plugins         []*Plugin
	Befores, Afters map[string][]string
	Pre, Post       func(state *SorterState) error
}

func (this Sorter) Sort(do func(state *SorterState, p *Plugin) (err error)) (result []*Plugin, err error) {
	var (
		graph = topsort.NewGraph()
		state = &SorterState{
			Plugins:    this.Plugins,
			Graph:      graph,
			pluginsMap: this.PluginsMap,
			Afters:     this.Afters,
			Befors:     this.Befores,
		}
	)
	log.Debug("sort")

	if state.Afters == nil {
		state.Afters = map[string][]string{}
	}
	if state.Befors == nil {
		state.Befors = map[string][]string{}
	}

	if this.Pre != nil {
		if err = this.Pre(state); err != nil {
			return
		}
	}

	for _, p := range this.Plugins {
		if err = do(state, p); err != nil {
			return
		}
		if !graph.ContainsNode(p.UID()) {
			graph.AddNode(p.UID())
		}
	}

	resultNames, err := graph.TopSort()
	if err != nil {
		return nil, errwrap.Wrap(err, "Top-Sort")
	}

	result = make([]*Plugin, len(resultNames))
	for i, uid := range resultNames {
		result[i] = state.pluginsMap[uid]
	}
	log.Debug("sort done")
	return
}
