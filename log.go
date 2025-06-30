package authkratos

var debugModeOpen = false

func GetDebugMode() bool {
	return debugModeOpen
}

func SetDebugMode(enable bool) {
	debugModeOpen = enable
}
