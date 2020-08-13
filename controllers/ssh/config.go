package ssh

import (
	"os"
	"path"
	"time"
)

type Config struct {
	User       string
	Host       string
	Port       int
	Password   string
	KeyFiles   []string
	Passphrase string

	StickySession bool
	// DisableAgentForwarding, if true, will not forward the SSH agent.
	DisableAgentForwarding bool

	// HandshakeTimeout limits the amount of time we'll wait to handshake before
	// saying the connection failed.
	HandshakeTimeout time.Duration

	// KeepAliveInterval sets how often we send a channel request to the
	// server. A value < 0 disables.
	KeepAliveInterval time.Duration

	// Timeout is how long to wait for a read or write to succeed.
	Timeout time.Duration
}

var DefaultConfig = &Config{
	Host: "localhost",
	Port: 22,
	User: "root",
	// KeyFiles: []string{path.Join(os.Getenv("HOME"), "/.ssh/id_rsa")},
}
var Default = DefaultConfig

func WithUser(user string) *Config {
	return Default.WithUser(user)
}
func (c *Config) WithUser(user string) *Config {
	if user == "" {
		user = "root"
	}
	c.User = user
	return c
}
func WithHost(host string) *Config {
	return Default.WithHost(host)
}

func (c *Config) WithHost(host string) *Config {
	if host == "" {
		host = "localhost"
	}
	c.Host = host
	return c
}

func WithPassword(password string) *Config {
	return Default.WithPassword(password)
}
func (c *Config) WithPassword(password string) *Config {
	c.Password = password
	return c
}

func WithKey(keyfile, passphrase string) *Config {
	return Default.WithKey(keyfile, passphrase)
}
func (c *Config) WithKey(keyfile, passphrase string) *Config {
	if keyfile == "" {
		if home := os.Getenv("HOME"); home != "" {
			keyfile = path.Join(home, "/.ssh/id_rsa")
		}
	}
	for _, s := range c.KeyFiles {
		if s == keyfile {
			return c
		}
	}
	c.KeyFiles = append(c.KeyFiles, keyfile)
	return c
}

//
func (c *Config) SetKeys(keyfiles []string) *Config {
	if keyfiles == nil {
		return c
	}
	t := make([]string, len(keyfiles))
	copy(t, keyfiles)
	c.KeyFiles = t
	return c
}
