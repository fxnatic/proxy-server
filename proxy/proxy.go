package proxy

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"proxy-server/auth"
	"proxy-server/config"
	"strings"

	"github.com/sirupsen/logrus"
)

type counterWriter struct {
	io.Writer
	count int64
}

func (cw *counterWriter) Write(p []byte) (int, error) {
	n, err := cw.Writer.Write(p)
	if err == nil {
		cw.count += int64(n)
	}
	return n, err
}

func HandleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		logrus.Errorf("Failed to read request: %v", err)
		return
	}

	authHeader := req.Header.Get("Proxy-Authorization")
	user, authenticated, aboveLimit := getUsernameAndAuth(authHeader)
	if !authenticated {
		conn.Write([]byte("HTTP/1.1 407 Proxy Authentication Required\r\n" +
			"Proxy-Authenticate: Basic realm=\"Restricted\"\r\n\r\n"))
		return
	}

	if aboveLimit {
		conn.Write([]byte("HTTP/1.1 429 Too Many Requests\r\n" +
			"Proxy-Authenticate: Basic realm=\"Restricted\"\r\n\r\n"))
		return
	}

	logrus.Infof("Handling request for URL: %s by user: %s", req.URL.String(), user)

	if req.Method == http.MethodConnect {
		handleConnect(conn, req, user)
	} else {
		proxyHTTP(conn, req, user)
	}

}

func getUsernameAndAuth(authHeader string) (string, bool, bool) {
	if authHeader == "" || !strings.HasPrefix(authHeader, "Basic ") {
		return "", false, false
	}

	decoded, err := base64.StdEncoding.DecodeString(authHeader[len("Basic "):])
	if err != nil {
		return "", false, false
	}

	credentials := string(decoded)
	parts := strings.Split(credentials, ":")
	if len(parts) != 2 {
		return "", false, false
	}

	data, exists := auth.GetUser(parts[0])
	if !exists {
		return parts[0], false, false
	}
	expectedCredentials := fmt.Sprintf("%s:%s", parts[0], data.Password)

	return parts[0], credentials == expectedCredentials, data.Usage > data.Limit
}

func handleConnect(conn net.Conn, req *http.Request, user string) {
	proxy := config.GetProxy()

	dialAddr := fmt.Sprintf("%s:%s", proxy.IP, proxy.Port)
	logrus.Infof("Dialing to proxy server at %s", dialAddr)

	proxyConn, err := net.Dial("tcp", dialAddr)
	if err != nil {
		logrus.Errorf("Failed to dial proxy: %v", err)
		httpError(conn, fmt.Errorf("failed to dial proxy: %v", err))
		return
	}
	defer proxyConn.Close()

	connectReq := &http.Request{
		Method: "CONNECT",
		URL:    &url.URL{Opaque: req.URL.Host},
		Host:   req.URL.Host,
		Header: make(http.Header),
	}

	if proxy.Username != "" && proxy.Password != "" {
		proxyAuth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", proxy.Username, proxy.Password)))
		connectReq.Header.Set("Proxy-Authorization", "Basic "+proxyAuth)
	}

	if err := connectReq.Write(proxyConn); err != nil {
		logrus.WithField("module", "proxy").Errorf("Failed to send CONNECT request: %v", err)
		httpError(conn, fmt.Errorf("failed to send CONNECT request: %v", err))
		return
	}

	br := bufio.NewReader(proxyConn)
	resp, err := http.ReadResponse(br, connectReq)
	if err != nil {
		httpError(conn, err)
		return
	}
	if resp.StatusCode != 200 {
		httpError(conn, fmt.Errorf("non-200 status code from proxy: %d", resp.StatusCode))
		return
	}

	conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	proxyConnWrite := counterWriter{Writer: proxyConn}
	clientConnWrite := counterWriter{Writer: conn}

	go func() {
		io.Copy(&proxyConnWrite, conn)
		logrus.Infof("User %s sent %d bytes to %s", user, proxyConnWrite.count, req.URL.Host)
		go auth.IncrUsage(user, proxyConnWrite.count)
	}()
	io.Copy(&clientConnWrite, proxyConn)
	logrus.Infof("User %s received %d bytes from %s", user, clientConnWrite.count, req.URL.Host)
	go auth.IncrUsage(user, clientConnWrite.count)
	proxyConn.Close()
	conn.Close()
}

func proxyHTTP(conn net.Conn, req *http.Request, user string) {
	proxy := config.GetProxy()
	proxyURL, err := url.Parse(fmt.Sprintf("http://%s:%s", proxy.IP, proxy.Port))
	if err != nil {
		httpError(conn, err)
		return
	}

	if proxy.Username != "" && proxy.Password != "" {
		credentials := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", proxy.Username, proxy.Password)))
		req.Header.Set("Proxy-Authorization", "Basic "+credentials)
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req.RequestURI = ""
	req.Header.Del("Proxy-Authorization")

	resp, err := client.Do(req)
	if err != nil {
		httpError(conn, err)
		return
	}
	defer resp.Body.Close()

	cw := counterWriter{Writer: conn}
	resp.Write(&cw)
	logrus.Infof("User %s sent %d bytes to %s", user, cw.count, req.RequestURI)
	go auth.IncrUsage(user, cw.count)
	conn.Close()
}

func httpError(conn net.Conn, err error) {
	logrus.Errorf("HTTP error: %v", err.Error())

	resp := &http.Response{
		StatusCode: http.StatusServiceUnavailable,
		Status:     "503 Service Unavailable",
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Body:       io.NopCloser(strings.NewReader("")),
	}
	resp.Write(conn)
}
