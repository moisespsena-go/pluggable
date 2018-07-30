package pluggable

import "github.com/moisespsena/go-edis"

type EventDispatcherInterface = edis.EventDispatcherInterface
type EventInterface = edis.EventInterface
type Event = edis.Event
type CallbackFunc = edis.CallbackFunc
type CallbackFuncE = edis.CallbackFuncE

var EAll = edis.EAll
