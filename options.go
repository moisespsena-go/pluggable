package module

import (
	"github.com/moisespsena/go-options"
)

type Options struct {
	options.Options
}

func NewOptions(data ...map[string]interface{}) *Options {
	return &Options{options.NewOptions(data...)}
}
