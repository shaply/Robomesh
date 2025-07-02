package shared

import "os"

var (
	DEBUG_MODE = false
)

func InitConfig() {
	DEBUG_MODE = os.Getenv("DEBUG") == "true"
}
