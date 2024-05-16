package config

import (
	"math/rand"
	"os"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
)

var Proxies []ProxyDetails
var proxyMutex sync.Mutex

type ProxyDetails struct {
	IP       string
	Port     string
	Username string
	Password string
}

func LoadProxies() {
	proxyData, err := os.ReadFile("proxies.txt")
	if err != nil {
		logrus.Fatalf("Failed to read proxy file: %v", err)
		return
	}

	proxyMutex.Lock()
	defer proxyMutex.Unlock()

	Proxies = nil

	proxyLines := strings.Split(string(proxyData), "\n")
	for _, line := range proxyLines {
		parts := strings.Split(line, ":")
		if len(parts) == 4 {
			Proxies = append(Proxies, ProxyDetails{
				IP:       parts[0],
				Port:     parts[1],
				Username: parts[2],
				Password: parts[3],
			})
		} else if len(parts) == 2 {
			Proxies = append(Proxies, ProxyDetails{
				IP:   parts[0],
				Port: parts[1],
			})
		}
	}
	logrus.Infof("Proxies reloaded successfully.")
}

func GetProxy() ProxyDetails {
	proxyMutex.Lock()
	defer proxyMutex.Unlock()
	return Proxies[rand.Intn(len(Proxies))]
}

func WatchProxies() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logrus.Fatalf("Error creating file watcher: %v", err)
	}
	defer watcher.Close()

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					logrus.Infof("Detected update in proxies.txt, reloading proxies.")
					LoadProxies()
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				logrus.Fatalf("Error watching proxies.txt file: %v", err)
			}
		}
	}()

	err = watcher.Add("proxies.txt")
	if err != nil {
		logrus.Fatalf("Error adding watcher on proxies.txt: %v", err)
	}
}

func SetupLogging() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:             true,
		ForceColors:               true,
		EnvironmentOverrideColors: true,
	})
}
