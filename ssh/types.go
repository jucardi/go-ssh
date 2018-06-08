package ssh

import (
	"fmt"
	"io"
	"net"

	"golang.org/x/crypto/ssh"
)

const (
	ProtocolTCP = NetworkProtocol("tcp")
	ProtocolUDP = NetworkProtocol("udp")
)

// IClient defines the contract for an SSH client.
type IClient interface {
	Dial(endpoint *Endpoint) error
	Execute(cmd string, stdCombined ...bool) (*Output, error)
	ExecuteAsync(cmd string, callback ...AsyncCallback)
	SetEnv(key, val string)
}

type ITunnel interface {
	Start() error
}

type iVirtualClient interface {
	dial(net, address string) (net.Conn, error)
	session() (*ssh.Session, error)
	setClient(*ssh.Client)
	initExec(stdCombined ...bool) (*ssh.Session, *Output, error)
	prepareOutput(stdCombined ...bool) (*ssh.Session, *Output, error)
	setEnv(key, val string)
}

// NetworkProtocol indicates the network protocol.
type NetworkProtocol string

// AsyncCallback defines an asynchronous callback to be called at the end of an
// asynchronous operation. An error arg will be passed if the operation ended
// unsuccessfully.
type AsyncCallback func(*Output, error)

// Output encapsulates the generated Stdout and Stderr from the SSH operation.
type Output struct {
	Stdout   io.Reader
	Stderr   io.Reader
	combined io.ReadWriter
}

// Combined returns the combined result of Stdout and Stderr (if it was enabled in the client)
func (o *Output) Combined() io.Reader {
	return o.combined
}

// Endpoint defines a client endpoint for SSH.
type Endpoint struct {
	// Server host address
	Host string
	// Server port
	Port int
}

// RemoteEndpoint defines the forwarding endpoint in the remote server.
type RemoteEndpoint struct {
	*Endpoint
	// When Tunneling, indicates if it should use the local URL to do the remote connection.
	// Useful when doing /etc/hosts manipulation
	UseLocal bool
}

func (endpoint *Endpoint) String() string {
	return fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port)
}

type nilReadWriter struct {
}

func (n *nilReadWriter) Read(p []byte) (int, error) {
	return 0, nil
}

func (n *nilReadWriter) Write(p []byte) (int, error) {
	return 0, nil
}

func newOutput(stdout, stderr io.Reader, combined ...io.ReadWriter) *Output {
	var comb io.ReadWriter
	if len(combined) > 0 && combined[0] != nil {
		comb = combined[0]
	} else {
		comb = &nilReadWriter{}
	}
	ret := &Output{
		Stdout:   stdout,
		Stderr:   stderr,
		combined: comb,
	}
	go io.Copy(ret.combined, stdout)
	go io.Copy(ret.combined, stderr)
	return ret
}
