package main

import (
	"net"
	"proxy-server/config"
	"proxy-server/proxy"

	"github.com/sirupsen/logrus"
)

var Port = "8888"

func main() {
	config.SetupLogging()
	config.LoadProxies()
	config.WatchProxies()

	listener, err := net.Listen("tcp", ":"+Port)
	if err != nil {
		logrus.Fatalf("Error starting TCP listener: %v", err)
	}
	defer listener.Close()

	logrus.Infof("Proxy server listening on :%s", Port)
	for {
		conn, err := listener.Accept()
		if err != nil {
			logrus.Errorf("Error accepting connection: %v", err)
			continue
		}

		go proxy.HandleConnection(conn)
	}
}
