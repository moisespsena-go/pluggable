package pluggable

type Extension interface {
	Init(plugins *Plugins) error
}
