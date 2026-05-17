package mcpclient

import (
	"fmt"
	"net/url"
)

var httpHostPolicy func(host string) bool

func SetHTTPHostPolicy(p func(host string) bool) {
	httpHostPolicy = p
}

func checkHTTPHostAllowed(host string) bool {
	if httpHostPolicy == nil {
		return true
	}
	return httpHostPolicy(host)
}

func validateHTTPMCPURL(label, raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%s: url: %w", label, err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%s: url: ожидается http или https", label)
	}

	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("%s: url: пустой хост", label)
	}

	if !checkHTTPHostAllowed(host) {
		return fmt.Errorf("%s: хост %q не разрешён политикой MCP", label, host)
	}

	return nil
}
