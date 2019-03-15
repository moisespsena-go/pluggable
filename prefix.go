package pluggable

import (
	"github.com/moisespsena/go-path-helpers"
	"github.com/op/go-logging"
)

var (
	PKG = path_helpers.GetCalledDir()
	log = logging.MustGetLogger(PKG)
	v = logging.DEBUG
)