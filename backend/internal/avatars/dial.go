package avatars

import (
	"net"
	"time"
)

var netDialer = net.Dialer{Timeout: 5 * time.Second}
