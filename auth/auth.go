package auth

import (
	"sync"

	"github.com/alecthomas/units"
)

type ProxyUser struct {
	Password string
	Usage    int64
	Limit    int64
}

var (
	ProxyUsers = map[string]ProxyUser{
		"admin": {
			Password: "password",
			Usage:    0,
			Limit:    int64(1 * units.GiB),
		},
		"user1": {
			Password: "password1",
			Usage:    0,
			Limit:    int64(1 * units.GiB),
		},
	}
	ProxyUserMutex = sync.RWMutex{}
)

func GetUser(username string) (ProxyUser, bool) {
	ProxyUserMutex.RLock()
	value, exists := ProxyUsers[username]
	ProxyUserMutex.RUnlock()
	return value, exists
}

func IncrUsage(username string, count int64) bool {
	ProxyUserMutex.RLock()
	ProxyUsers[username] = ProxyUser{
		Password: ProxyUsers[username].Password,
		Usage:    ProxyUsers[username].Usage + count,
	}
	ProxyUserMutex.RUnlock()
	return true
}
