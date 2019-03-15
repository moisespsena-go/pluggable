package pluggable

func OnPostInit(dis EventDispatcherInterface, callbacks ...interface{}) error {
	return dis.OnE(E_POST_INIT, callbacks...)
}

func OnInit(dis EventDispatcherInterface, callbacks ...interface{}) error {
	return dis.OnE(E_INIT, callbacks...)
}
