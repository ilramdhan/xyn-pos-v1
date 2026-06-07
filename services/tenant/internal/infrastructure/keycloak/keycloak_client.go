package keycloak

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client wraps Keycloak Admin REST API and OIDC token introspection.
type Client struct {
	baseURL      string
	realm        string
	clientID     string
	clientSecret string
	httpClient   *http.Client
}

// Config holds Keycloak connection settings.
type Config struct {
	BaseURL      string
	Realm        string
	ClientID     string
	ClientSecret string
}

// NewClient creates a Keycloak client with a 10s HTTP timeout.
func NewClient(cfg Config) *Client {
	return &Client{
		baseURL:      strings.TrimRight(cfg.BaseURL, "/"),
		realm:        cfg.Realm,
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

// IntrospectResult is the parsed Keycloak token introspection response.
type IntrospectResult struct {
	Active  bool   `json:"active"`
	Subject string `json:"sub"`
	Email   string `json:"email"`
}

// Introspect verifies a Keycloak access token. Returns error if inactive or invalid.
func (c *Client) Introspect(ctx context.Context, accessToken string) (*IntrospectResult, error) {
	endpoint := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token/introspect",
		c.baseURL, c.realm)

	form := url.Values{}
	form.Set("token", accessToken)
	form.Set("client_id", c.clientID)
	form.Set("client_secret", c.clientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint,
		strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("keycloak.Introspect build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("keycloak.Introspect http: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("keycloak.Introspect status=%d body=%s", resp.StatusCode, body)
	}

	var result IntrospectResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("keycloak.Introspect decode: %w", err)
	}

	if !result.Active {
		return nil, fmt.Errorf("keycloak.Introspect: token is inactive or expired")
	}

	return &result, nil
}

// CreateUserRequest is the payload for creating a Keycloak user.
type CreateUserRequest struct {
	Email     string
	Password  string
	FirstName string
}

// CreateUser creates a new user in Keycloak and returns their subject ID.
// adminToken is a Keycloak admin access token (obtained separately).
func (c *Client) CreateUser(ctx context.Context, adminToken string, req CreateUserRequest) (string, error) {
	endpoint := fmt.Sprintf("%s/admin/realms/%s/users", c.baseURL, c.realm)

	body, _ := json.Marshal(map[string]any{
		"email":     req.Email,
		"username":  req.Email,
		"firstName": req.FirstName,
		"enabled":   true,
		"credentials": []map[string]any{
			{"type": "password", "value": req.Password, "temporary": false},
		},
	})

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint,
		strings.NewReader(string(body)))
	if err != nil {
		return "", fmt.Errorf("keycloak.CreateUser build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+adminToken)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("keycloak.CreateUser http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("keycloak.CreateUser status=%d body=%s", resp.StatusCode, b)
	}

	location := resp.Header.Get("Location")
	parts := strings.Split(location, "/")
	if len(parts) == 0 {
		return "", fmt.Errorf("keycloak.CreateUser: missing Location header")
	}
	return parts[len(parts)-1], nil
}
