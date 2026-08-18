package storage

import (
	"strconv"

	"mc-server-checker/internal/domain"
)

const (
	addressKey = "last_successful_address"
	portKey    = "last_successful_port"
)

type StringPreferences interface {
	StringWithFallback(key, fallback string) string
	SetString(key, value string)
}

type Preferences struct {
	values StringPreferences
}

func NewPreferences(values StringPreferences) *Preferences {
	return &Preferences{values: values}
}

func (p *Preferences) Load() (address, port string) {
	if p == nil || p.values == nil {
		return "", ""
	}
	return p.values.StringWithFallback(addressKey, ""), p.values.StringWithFallback(portKey, "")
}

func (p *Preferences) Save(target domain.Target) {
	if p == nil || p.values == nil || target.Host == "" || target.Port == 0 {
		return
	}
	p.values.SetString(addressKey, target.Host)
	port := ""
	if !target.UseSRV {
		port = strconv.FormatUint(uint64(target.Port), 10)
	}
	p.values.SetString(portKey, port)
}
