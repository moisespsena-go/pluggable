package pluggable

func EPostInit(dis EventDispatcherInterface, callbacks ...interface{}) error {
	return dis.OnE(E_POST_INIT, callbacks...)
}

func EInit(dis EventDispatcherInterface, callbacks ...interface{}) error {
	return dis.OnE(E_INIT, callbacks...)
}
