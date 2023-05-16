package limiter

import (
	"sync"
)

type ConnLimiter struct {
	realTime  bool
	ipLimit   int
	connLimit int
	count     sync.Map // map[string]int
	ip        sync.Map // map[string]map[string]int
}

func NewConnLimiter(conn int, ip int) *ConnLimiter {
	return &ConnLimiter{
		connLimit: conn,
		ipLimit:   ip,
		count:     sync.Map{},
		ip:        sync.Map{},
	}
}

func (c *ConnLimiter) AddConnCount(user string, ip string, isTcp bool) (limit bool) {
	if c.connLimit != 0 {
		if v, ok := c.count.Load(user); ok {
			if v.(int) >= c.connLimit {
				return true
			} else if isTcp { // tcp protocol
				c.count.Store(user, v.(int)+1)
			}
		} else if isTcp { // tcp protocol
			c.count.Store(user, 1)
		}
	}
	if c.ipLimit == 0 {
		return false
	}
	// default user map
	ipMap := new(sync.Map)
	if isTcp {
		ipMap.Store(ip, 2)
	} else {
		ipMap.Store(ip, 1)
	}
	// check user online ip
	if v, ok := c.ip.LoadOrStore(user, ipMap); ok {
		// have user
		ips := v.(*sync.Map)
		cn := 0
		if online, ok := ips.Load(ip); ok {
			// online ip
			if isTcp {
				// count add
				ips.Store(ip, online.(int)+2)
			}
		} else {
			// not online ip
			ips.Range(func(_, _ interface{}) bool {
				cn++
				if cn >= c.ipLimit {
					limit = true
					return false
				}
				return true
			})
			if limit {
				return
			}
			if isTcp {
				ips.Store(ip, 2)
			} else {
				ips.Store(ip, 1)
			}
		}
	}
	return
}

// DelConnCount Delete tcp connection count, no tcp do not use
func (c *ConnLimiter) DelConnCount(user string, ip string) {
	if c.connLimit != 0 {
		if v, ok := c.count.Load(user); ok {
			if v.(int) == 1 {
				c.count.Delete(user)
			} else {
				c.count.Store(user, v.(int)-1)
			}
		}
	}
	if c.ipLimit == 0 {
		return
	}
	if i, ok := c.ip.Load(user); ok {
		is := i.(*sync.Map)
		if i, ok := is.Load(ip); ok {
			if i.(int) == 2 {
				is.Delete(ip)
			} else {
				is.Store(user, i.(int)-2)
			}
			notDel := false
			c.ip.Range(func(_, _ any) bool {
				notDel = true
				return false
			})
			if !notDel {
				c.ip.Delete(user)
			}
		}
	}
}

// ClearPacketOnlineIP Clear udp,icmp and other packet protocol online ip
func (c *ConnLimiter) ClearPacketOnlineIP() {
	c.ip.Range(func(_, v any) bool {
		userIp := v.(*sync.Map)
		userIp.Range(func(ip, v any) bool {
			if v.(int) == 1 {
				userIp.Delete(ip)
			}
			return true
		})
		return true
	})
}
