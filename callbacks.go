package pluggable

type PluginEventCallbackInterface interface {
	Call(pe PluginEventInterface) error
}

type PluginCallbackFuncE func(pe PluginEventInterface) error

func (p PluginCallbackFuncE) Call(pe PluginEventInterface) error {
	return p(pe)
}

type PluginCallbackFunc func(pe PluginEventInterface)

func (p PluginCallbackFunc) Call(pe PluginEventInterface) error {
	p(pe)
	return nil
}

type PluginEventCallbackE func(e PluginEventInterface) error

func (c PluginEventCallbackE) Call(e EventInterface) error {
	return c(e.(PluginEventInterface))
}

type PluginEventCallback func(e PluginEventInterface)

func (c PluginEventCallback) Call(e EventInterface) error {
	c(e.(PluginEventInterface))
	return nil
}
