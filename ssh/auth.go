package ssh

import (
	"io/ioutil"
	"net"

	"github.com/jucardi/go-logger-lib/log"
	"golang.org/x/crypto/ssh"
)

type Auth struct {
	PrivateKeyFile string
	User           string
	Password       string
}

func (k *Auth) PublicKeyAuth() ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(k.PrivateKeyFile)
	if err != nil {
		log.Panicf("Unable to read key file, %v", err)
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		log.Panicf("Unable to parse private key, %v", err)
	}
	return ssh.PublicKeys(key)
}

func (k *Auth) ClientConfig() *ssh.ClientConfig {
	ret := &ssh.ClientConfig{
		User:            k.User,
		Auth:            []ssh.AuthMethod{},
		HostKeyCallback: hostKeyCallback,
	}

	if k.PrivateKeyFile != "" {
		ret.Auth = append(ret.Auth, k.PublicKeyAuth())
	}

	if k.Password != "" {
		ret.Auth = append(ret.Auth, ssh.Password(k.Password))
	}

	return ret
}

func hostKeyCallback(hostname string, remote net.Addr, key ssh.PublicKey) error {
	log.Debugf("-- callback -- Hostname: %v, Remote: %v, Key: %v", hostname, remote, key)
	return nil
}
