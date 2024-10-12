package catbox

import "github.com/sirupsen/logrus"

type state int

const (
	StateRunning state = iota
	StatePaused
	StateStopped
)

var G_config Config
var G_logger = logrus.New()
var G_proxyManager *ProxyManager
var G_state state
