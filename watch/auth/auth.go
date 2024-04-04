package auth

import (
	"context"
	"net/http"
	"net/url"
)

type Method interface {
	GetUsername() string
	AuthenticatedRequest(cli *http.Client, req *http.Request) (*http.Response, error)
	Token(ctx context.Context, baseURL *url.URL, cli *http.Client) (string, error)
}
