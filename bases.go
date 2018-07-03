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
