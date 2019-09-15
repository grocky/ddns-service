package pubip

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// Authority Used to engage with public ip authorities.
type Authority struct {
	httpClient *http.Client
	domain     string
}

// NewAuthority Creates a new Authority
func NewAuthority(domain string) *Authority {
	return &Authority{
		domain: domain,
		httpClient: &http.Client{
			Timeout: time.Second * 2,
		},
	}
}

func (a *Authority) requestExternalIP() (string, error) {
	req, err := http.NewRequest("GET", a.domain, nil)
	if err != nil {
		return "", fmt.Errorf("Unable to create the request %s", err.Error())
	}
	req.Header.Set("User-Agent", "grocky: pubip")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("Request error: %s", err.Error())
	}

	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Unable to read body: %s", err.Error())
	}

	return string(b), nil
}
