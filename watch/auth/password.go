package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Password struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`

	token    string    `yaml:"-"`
	lastAuth time.Time `yaml:"-"`
}

type authRequest struct {
	Email    string `json:"email,omitempty"`
	Password string `json:"password,omitempty"`
}

func (p *Password) GetUsername() string {
	return p.Username
}

func (p *Password) Token(ctx context.Context, baseURL *url.URL, cli *http.Client) (string, error) {
	err := p.ObtainToken(ctx, baseURL, cli)
	if err != nil {
		return "", nil
	}
	return p.token, nil
}

func (p *Password) ObtainToken(ctx context.Context, baseURL *url.URL, cli *http.Client) error {
	data, err := json.Marshal(authRequest{
		Email:    p.Username,
		Password: p.Password,
	})

	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL.ResolveReference(&url.URL{
		Path: "/api/auth/signin",
	}).String(), bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")

	resp, err := cli.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	type AuthResponse struct {
		Token string `json:"token"`
	}

	var tokenResp AuthResponse
	respData, _ := io.ReadAll(resp.Body)
	err = json.NewDecoder(bytes.NewBuffer(respData)).Decode(&tokenResp)
	if err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if tokenResp.Token == "" {
		return fmt.Errorf("failed to auth to server %s", baseURL.String())
	}
	p.token = tokenResp.Token
	p.lastAuth = time.Now()

	return nil
}

func (p *Password) AuthenticatedRequest(cli *http.Client, req *http.Request) (*http.Response, error) {
	return p.authenticatedRequest(cli, req, true)
}

func (p *Password) authenticatedRequest(cli *http.Client, req *http.Request, retry bool) (*http.Response, error) {
	if time.Since(p.lastAuth) > time.Hour {
		err := p.ObtainToken(req.Context(), req.URL, cli)
		if err != nil {
			return nil, fmt.Errorf("failed to obtain auth token: %w", err)
		}
	}

	req.Header.Set("X-Token", p.token)
	req.Header.Set("X-Username", p.Username)
	resp, err := cli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	if resp.StatusCode == http.StatusUnauthorized && retry {
		err := p.ObtainToken(req.Context(), req.URL, cli)
		if err != nil {
			return nil, fmt.Errorf("failed to obtain auth token: %w", err)
		}
		return p.authenticatedRequest(cli, req, false)
	}

	return resp, nil
}
