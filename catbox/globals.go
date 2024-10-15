package catbox

import (
	"sync/atomic"

	"github.com/disgoorg/disgo/webhook"
	"github.com/sirupsen/logrus"
)

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
var G_webhook_client webhook.Client
var G_Req_Per_Sec atomic.Int64
var G_Found_Per_Min atomic.Int64
