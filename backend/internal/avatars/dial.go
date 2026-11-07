package avatars

import (
	"net"
	"syscall"
	"time"
)

var netDialer = net.Dialer{Timeout: 5 * time.Second}

// RestrictDial sets the Control func of the LDAP dialer (demo mode: only allowed addresses).
func RestrictDial(control func(network, address string, c syscall.RawConn) error) {
	netDialer.Control = control
}
