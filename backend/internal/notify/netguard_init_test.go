package notify

import "github.com/Rombond/tipsarr/backend/internal/netguard"

// httptest receivers listen on loopback, which the outbound policy always refuses in production.
func init() { netguard.AllowLoopbackForTests() }
