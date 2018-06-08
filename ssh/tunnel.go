package ssh

import (
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/jucardi/go-logger-lib/log"
	"golang.org/x/crypto/ssh"
)

// Tunnel defines an SSH tunnel
type Tunnel struct {
	Auth   *Auth
	Local  *Endpoint
	Server *Endpoint
	Remote *RemoteEndpoint

	listener   net.Listener
	serverConn *ssh.Client
	remoteConn net.Conn
}

// Start dials the required connections and starts the tunnel
func (t *Tunnel) Start() error {
	listener, err := t.dial()
	if err != nil {
		return fmt.Errorf("unable to establish connections: %v", err)
	}
	defer listener.Close()

	for {
		addr := listener.Addr()
		conn, err := listener.Accept()

		log.Debugf("listener addr     %s, network %s", addr.String(), addr.Network())
		log.Debugf("local connection  %s, network %s", conn.LocalAddr().String(), conn.LocalAddr().Network())
		log.Debugf("remote connection %s, network %s", conn.RemoteAddr().String(), conn.RemoteAddr().Network())

		if err != nil {
			log.Errorf("error accepting connections, %v", err)
			return err
		}
		log.Debug("client accepted")
		go t.forward(conn)
	}
}

func (t *Tunnel) dial() (net.Listener, error) {
	log.Debugf("starting listener %s", t.Local.String())

	listener, err := net.Listen("tcp", t.Local.String())
	if err != nil {
		return nil, fmt.Errorf("error starting listener, %v", err)
	}

	serverConn, err := ssh.Dial("tcp", t.Server.String(), t.Auth.ClientConfig())
	if err != nil {
		return nil, fmt.Errorf("server dial error: %s", err)
	}

	t.listener = listener
	t.serverConn = serverConn

	return listener, nil
}

func (t *Tunnel) forward(current net.Conn) {
	rem := t.Remote
	if rem.UseLocal {
		rem = &RemoteEndpoint{
			Endpoint: &Endpoint{
				Host: strings.Split(current.LocalAddr().String(), ":")[0],
				Port: rem.Port,
			},
		}
	}
	log.Debugf("fw to %s", rem.String())
	remote, err := t.serverConn.Dial("tcp", rem.String())

	if err != nil {
		log.Errorf("Remote dial error: %s\n", err)
		return
	}

	context := &connectionContext{
		current: current,
		remote:  remote,
	}

	go t.copy(current, remote, context.markCurrentToRemoteDone)
	go t.copy(remote, current, context.markRemoteToCurrentDone)
}

func (t *Tunnel) copy(writer, reader net.Conn, doneCallback func()) {
	if _, err := io.Copy(writer, reader); err != nil {
		log.Errorf("io.Copy error: %s", err)
	}
	doneCallback()
}

type connectionContext struct {
	current             net.Conn
	remote              net.Conn
	currentToRemoteDone bool
	remoteToCurrentDone bool
}

func (c *connectionContext) markCurrentToRemoteDone() {
	c.currentToRemoteDone = true
	c.closeIfDone()
}

func (c *connectionContext) markRemoteToCurrentDone() {
	c.remoteToCurrentDone = true
	c.closeIfDone()
}

func (c *connectionContext) closeIfDone() {
	if !c.readyToClose() {
		return
	}

	errC := c.current.Close()
	errR := c.remote.Close()

	if errC != nil || errR != nil {
		log.Errorf("errors while closing connections.\nCurrentErr: %v\nRemoteErr: %v", errC, errR)
	}
}

func (c *connectionContext) readyToClose() bool {
	return c.currentToRemoteDone && c.remoteToCurrentDone
}
