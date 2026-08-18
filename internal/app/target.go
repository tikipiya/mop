package app

import (
	"errors"
	"net"
	"strconv"
	"strings"

	"mc-server-checker/internal/domain"
)

var (
	ErrAddressRequired = errors.New("address is required")
	ErrInvalidAddress  = errors.New("address is invalid")
	ErrInvalidPort     = errors.New("port must be between 1 and 65535")
)

func NormalizeTarget(address, port string) (domain.Target, *domain.AppError) {
	host, err := normalizeHost(address)
	if err != nil {
		return domain.Target{}, validationError(err)
	}
	normalizedPort, err := normalizePort(port)
	if err != nil {
		return domain.Target{}, validationError(err)
	}
	return domain.Target{Host: host, Port: normalizedPort}, nil
}

func normalizeHost(address string) (string, error) {
	host := strings.TrimSpace(address)
	if host == "" {
		return "", ErrAddressRequired
	}

	if strings.HasPrefix(host, "[") || strings.HasSuffix(host, "]") {
		if !strings.HasPrefix(host, "[") || !strings.HasSuffix(host, "]") || len(host) < 3 {
			return "", ErrInvalidAddress
		}
		inner := host[1 : len(host)-1]
		ip := net.ParseIP(inner)
		if ip == nil || ip.To4() != nil {
			return "", ErrInvalidAddress
		}
		return ip.String(), nil
	}
	if strings.ContainsAny(host, "[]:") {
		return "", ErrInvalidAddress
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.To4() == nil {
			return "", ErrInvalidAddress
		}
		return ip.String(), nil
	}
	if !validHostname(host) {
		return "", ErrInvalidAddress
	}
	return host, nil
}

func normalizePort(port string) (uint16, error) {
	value := strings.TrimSpace(port)
	if value == "" {
		return domain.DefaultPort, nil
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0, ErrInvalidPort
		}
	}
	parsed, err := strconv.ParseUint(value, 10, 16)
	if err != nil || parsed == 0 {
		return 0, ErrInvalidPort
	}
	return uint16(parsed), nil
}

func validHostname(host string) bool {
	name := strings.TrimSuffix(host, ".")
	if name == "" || len(name) > 253 {
		return false
	}
	for _, label := range strings.Split(name, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, r := range label {
			if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' {
				continue
			}
			return false
		}
	}
	return true
}

func validationError(cause error) *domain.AppError {
	return &domain.AppError{
		Kind:      domain.ErrorValidation,
		Message:   "入力内容を確認してください。",
		Cause:     cause,
		Retryable: false,
	}
}
