package pluggable

import (
	"github.com/moisespsena/go-default-logger"
	"github.com/moisespsena/go-path-helpers"
	"github.com/op/go-logging"
)

var PREFIX string

var log *logging.Logger

func init() {
	PREFIX = path_helpers.GetCalledDir()
	log = defaultlogger.NewLogger(PREFIX)
}
