package pluggable

import "github.com/moisespsena-go/edis"

type (
	EventDispatcherInterface = edis.EventDispatcherInterface
	EventInterface           = edis.EventInterface
	Event                    = edis.Event
	CallbackFunc             = edis.CallbackFunc
	CallbackFuncE            = edis.CallbackFuncE
)

var (
	EAll     = edis.EAll
	NewEvent = edis.NewEvent
)
