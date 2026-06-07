package cache

import "errors"

// ErrInvalidAddr is returned when Addr is empty.
var ErrInvalidAddr = errors.New("cache: addr cannot be empty")

// Config holds Redis connection settings.
type Config struct {
	Addr       string
	Password   string
	DB         int
	MaxRetries int
}

// Validate returns ErrInvalidAddr if Addr is empty.
func (c Config) Validate() error {
	if c.Addr == "" {
		return ErrInvalidAddr
	}
	return nil
}
