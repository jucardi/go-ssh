package ssh

import "time"

// Config encapsulates the configuration parameters for an SSH client.
type Config struct {
	// DialRetryTimeout is the sleep time between dial retries.
	DialRetryTimeout time.Duration `json:"dial_retry_timeout"`

	// DialMaxRetries is the max attempts to retry to establish a client.
	// -1 for indefinite.
	DialMaxRetries int `json:"dial_max_retries"`
}
