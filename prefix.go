package pluggable

import (
	"github.com/moisespsena-go/logging"
	path_helpers "github.com/moisespsena-go/path-helpers"
)

var (
	PKG = path_helpers.GetCalledDir()
	log = logging.GetOrCreateLogger(PKG)
)
