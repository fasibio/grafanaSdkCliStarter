package grafanasdkclistarter

import (
	"net/http"
)

type BasicAuthTransport struct {
	credentials BasicAuth
}

type BasicAuth struct {
	Username string
	Password string
}

func (bat BasicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(bat.credentials.Username, bat.credentials.Password)
	req.Header.Set("Content-Type", "application/json")
	return http.DefaultTransport.RoundTrip(req)
}
