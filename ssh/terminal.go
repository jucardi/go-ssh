package ssh

import (
	"fmt"
	"github.com/jucardi/go-logger-lib/log"
	"golang.org/x/crypto/ssh"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const (
	width, height = 80, 40
)

type virtualTerminalImpl struct {
	*virtualClientImpl
}

func NewTerminal(auth *Auth, config *Config) IClient {
	return &client{
		auth:   auth,
		config: config,
		iVirtualClient: &virtualTerminalImpl{
			virtualClientImpl: &virtualClientImpl{
				env: map[string]string{},
			},
		},
	}
}

func (t *virtualTerminalImpl) initExec(stdCombined ...bool) (*ssh.Session, *Output, error) {
	s, o, err := t.prepareOutput(stdCombined...)

	if err != nil {
		return nil, nil, err
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	w, h := getSize()
	if err := s.RequestPty("xterm", h, w, modes); err != nil {
		return nil, nil, fmt.Errorf("request for pseudo terminal failed: %s", err)
	}

	stdin, err := s.StdinPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to setup stdin for session: %v", err)
	}

	go io.Copy(stdin, os.Stdin)
	go io.Copy(os.Stdout, o.Stdout)
	go io.Copy(os.Stderr, o.Stderr)

	return s, o, nil
}

func getSize() (int, int) {
	cmd := exec.Command("stty", "size")
	out, err := cmd.Output()
	if err != nil {
		log.Debug("failed to obtain terminal size, returning default values")
		return width, height
	}

	split := strings.Split(string(out), " ")
	w, ew := strconv.Atoi(split[0])
	h, eh := strconv.Atoi(split[1])

	if ew != nil || eh != nil {
		log.Debug("failed to parse terminal size, returning default values")
		return width, height
	}

	return w, h
}
