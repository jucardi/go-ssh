package ssh

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/jucardi/go-logger-lib/log"
	"golang.org/x/crypto/ssh"
)

// HttpTunnel defines an SSH tunnel and forwards the local and remote connection using http protocol
type HttpTunnel struct {
	Auth   *Auth
	Local  *Endpoint
	Server *Endpoint
	Remote *RemoteEndpoint

	listener   net.Listener
	serverConn *ssh.Client
	remoteConn net.Conn
}

// Start dials the required connections and starts the tunnel
func (t *HttpTunnel) Start() error {
	err := t.dial()
	if err != nil {
		return fmt.Errorf("unable to establish connections: %v", err)
	}

	return http.ListenAndServe(fmt.Sprintf(":%d", t.Local.Port), t)
}

func (t *HttpTunnel) dial() error {
	log.Debugf("starting listener %s", t.Local.String())

	serverConn, err := ssh.Dial("tcp", t.Server.String(), t.Auth.ClientConfig())
	if err != nil {
		return fmt.Errorf("server dial error: %s", err)
	}

	t.serverConn = serverConn
	return nil
}

func (t *HttpTunnel) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	host := strings.Split(req.Host, ":")[0]

	log.Debugf("received connection, host: %s | remoteAddr: %s", host, req.RemoteAddr)

	rem := t.Remote
	if rem.UseLocal {
		rem = &RemoteEndpoint{
			Endpoint: &Endpoint{
				Host: host,
				Port: rem.Port,
			},
		}
	}
	log.Debugf("fw to %s", rem.String())

	remote, err := t.serverConn.Dial("tcp", rem.String())

	if err != nil {
		log.Errorf("remote dial error: %s\n", err)
		return
	}

	go func() {
		if err := req.WriteProxy(remote); err != nil {
			log.Errorf("error writing to proxy", err)
		}
	}()

	go func() {
		if _, err := io.Copy(resp, remote); err != nil {
			log.Errorf("io.Copy error: %s", err)
		}
	}()
}

func (t *HttpTunnel) RoundTrip(req *http.Request) (*http.Response, error) {
	host := strings.Split(req.Host, ":")[0]

	log.Debugf("received connection, host: %s | remoteAddr: %s", host, req.RemoteAddr)

	rem := t.Remote
	if rem.UseLocal {
		rem = &RemoteEndpoint{
			Endpoint: &Endpoint{
				Host: host,
				Port: rem.Port,
			},
		}
	}
	log.Debugf("fw to %s", rem.String())

	remote, err := t.serverConn.Dial("tcp", rem.String())

	if err != nil {
		log.Errorf("remote dial error: %s\n", err)
		return nil, err
	}
	var b bytes.Buffer
	writer := bufio.NewWriter(&b)

	chDone := make(chan bool)
	go func() {
		if err := req.WriteProxy(remote); err != nil {
			log.Errorf("error writing to proxy", err)
		}
		chDone <- true
	}()

	go func() {
		if _, err := io.Copy(writer, remote); err != nil {
			log.Errorf("io.Copy error: %s", err)
		}
		chDone <- true
	}()

	<-chDone
	closer := &closerImpl{
		Buffer: b,
		remote: remote,
	}
	resp := &http.Response{}
	resp.Body = closer
	return resp, nil
}

type closerImpl struct {
	bytes.Buffer
	remote net.Conn
}

func (c *closerImpl) Close() error {
	return c.remote.Close()
}

// createReverseProxy creates a reverse proxy on anything that falls through the global
// route / fall-through. This ultimately tunnels the RESTful call to the remote
// app/microservice to bypass CORS.
//
func (t *HttpTunnel) createReverseProxy() *httputil.ReverseProxy {
	director := func(req *http.Request) {
		req.URL.Scheme = "http"
	}
	proxy := &httputil.ReverseProxy{
		Director:  director,
		Transport: t,
	}
	return proxy
}
