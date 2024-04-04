package auth

import (
	"context"
	"net/http"
	"net/url"
)

type Token struct {
	Username  string `yaml:"username"`
	AuthToken string `yaml:"token"`
}

func (p *Token) GetUsername() string {
	return p.Username
}

func (t *Token) Token(_ context.Context, _ *url.URL, _ *http.Client) (string, error) {
	return t.AuthToken, nil
}

func (t *Token) AuthenticatedRequest(cli *http.Client, req *http.Request) (*http.Response, error) {
	req.Header.Set("X-Token", t.AuthToken)
	req.Header.Set("X-Username", t.Username)
	return cli.Do(req)
}
