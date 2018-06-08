package ssh

import (
	"bytes"
	"fmt"
	"github.com/prometheus/common/log"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
)

type client struct {
	iVirtualClient
	auth   *Auth
	config *Config
}

func NewClient(auth *Auth, config *Config) IClient {
	return &client{
		auth:   auth,
		config: config,
		iVirtualClient: &virtualClientImpl{
			env: map[string]string{},
		},
	}
}

func (c *client) Dial(endpoint *Endpoint) error {
	conn, err := ssh.Dial("tcp", endpoint.String(), c.auth.ClientConfig())
	if err != nil {
		return fmt.Errorf("failed to dial: %s", err)
	}
	c.setClient(conn)
	return nil
}

func (c *client) DialFromConnection(net, address string) (net.Conn, error) {
	return c.dial(net, address)
}

func (c *client) Execute(cmd string, stdCombined ...bool) (*Output, error) {
	s, o, err := c.initExec(stdCombined...)
	if err != nil {
		return nil, err
	}
	defer s.Close()
	err = c.execute(cmd, s)

	return o, err
}

func (c *client) ExecuteAsync(cmd string, callback ...AsyncCallback) {
	go func() {
		result, err := c.Execute(cmd)
		for _, cb := range callback {
			cb(result, err)
		}
	}()
}

func (c *client) SetEnv(key, val string) {
	c.setEnv(key, val)
}

func (c *client) execute(cmd string, s *ssh.Session) error {
	if e := s.Run(cmd); e != nil {
		if val, ok := e.(*ssh.ExitError); ok {
			return makeErr(val.String(), val.ExitStatus(), e)
		}
		if e, ok := e.(*ssh.ExitMissingError); ok {
			return e
		}
		return fmt.Errorf("an error occurred while executing the command, %v", e)
	}
	return nil
}

type virtualClientImpl struct {
	client *ssh.Client
	env    map[string]string
}

func (a *virtualClientImpl) session() (*ssh.Session, error) {
	if a.client == nil {
		return nil, ErrNoConnection
	}
	ret, err := a.client.NewSession()
	if err != nil {
		return nil, err
	}

	for k, v := range a.env {
		if err = ret.Setenv(k, v); err != nil {
			log.Warnf("failed to set environment variable '%s': %v", k, err)
		}
	}
	return ret, nil
}

func (a *virtualClientImpl) dial(net, address string) (net.Conn, error) {
	if a.client == nil {
		return nil, ErrNoConnection
	}
	return a.client.Dial(net, address)
}

func (a *virtualClientImpl) setClient(client *ssh.Client) {
	a.client = client
}

func (a *virtualClientImpl) initExec(stdCombined ...bool) (*ssh.Session, *Output, error) {
	return a.prepareOutput(stdCombined...)
}

func (a *virtualClientImpl) prepareOutput(stdCombined ...bool) (*ssh.Session, *Output, error) {
	s, err := a.session()

	if err != nil {
		return nil, nil, err
	}

	stderr, err := s.StderrPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to setup stdin for session: %v", err)
	}

	stdout, err := s.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to setup stdout for session: %v", err)
	}

	var combined io.ReadWriter
	if len(stdCombined) > 0 && stdCombined[0] {
		combined = &bytes.Buffer{}
	}
	o := newOutput(stdout, stderr, combined)
	return s, o, nil
}

func (a *virtualClientImpl) setEnv(key, val string) {
	a.env[key] = val
}
