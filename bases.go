package pluggable

import "github.com/op/go-logging"

type GlobalOptions struct {
	GlobalOptions *Options
}

func (b *GlobalOptions) SetGlobalOptions(options *Options) {
	b.GlobalOptions = options
}

func (b GlobalOptions) GetGlobalOptions() *Options {
	return b.GlobalOptions
}

type Logged struct {
	log *logging.Logger
}

func (l *Logged) SetLog(log *logging.Logger) {
	l.log = log
}

func (l *Logged) Log() *logging.Logger {
	return l.log
}

type Accessible struct {
	plugin *Plugin
}

func (a *Accessible) Plugin() *Plugin {
	return a.plugin
}

func (a *Accessible) SetPlugin(plugin *Plugin) {
	a.plugin = plugin
}
