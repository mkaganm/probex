package auth

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mkaganm/probex/internal/models"
)

// commonAuthEndpoints are well-known authentication endpoint paths to probe.
var commonAuthEndpoints = []string{
	"/auth",
	"/login",
	"/oauth/token",
	"/api/auth",
	"/api/login",
	"/oauth/authorize",
	"/token",
	"/api/token",
}

// Detector identifies authentication requirements of an API.
type Detector struct {
	baseURL string
	client  *http.Client
}

// New creates a new auth Detector.
func New(baseURL string) *Detector {
	return &Detector{
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
	}
}

// Detect probes the API to determine authentication requirements.
func (d *Detector) Detect(ctx context.Context) (*models.AuthInfo, error) {
	// Step 1: Make a GET to the base URL and check for 401/403.
	authRequired, wwwAuth, err := d.probeBaseURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("probing base URL: %w", err)
	}

	// If no auth required, return none.
	if !authRequired {
		return &models.AuthInfo{
			Type: models.AuthNone,
		}, nil
	}

	// Step 2: Parse WWW-Authenticate header if present.
	if wwwAuth != "" {
		info := parseWWWAuthenticate(wwwAuth)
		if info != nil {
			return info, nil
		}
	}

	// Step 3: Probe common auth endpoints to detect OAuth2 or login flows.
	info, err := d.probeAuthEndpoints(ctx)
	if err != nil {
		// Non-fatal; fall through to default.
		_ = err
	}
	if info != nil {
		return info, nil
	}

	// Step 4: Default — auth is required but type is unclear; assume API key in header.
	return &models.AuthInfo{
		Type:     models.AuthAPIKey,
		Location: "header",
		Key:      "Authorization",
	}, nil
}

// probeBaseURL sends a GET to the base URL and checks for auth-related status codes.
// Returns whether auth is required, the WWW-Authenticate header value (if any), and an error.
func (d *Detector) probeBaseURL(ctx context.Context) (bool, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.baseURL, nil)
	if err != nil {
		return false, "", err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return false, "", err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		wwwAuth := resp.Header.Get("WWW-Authenticate")
		return true, wwwAuth, nil
	default:
		return false, "", nil
	}
}

// parseWWWAuthenticate parses the WWW-Authenticate header to determine the auth type.
func parseWWWAuthenticate(header string) *models.AuthInfo {
	lower := strings.ToLower(header)

	switch {
	case strings.HasPrefix(lower, "bearer"):
		info := &models.AuthInfo{
			Type:     models.AuthBearer,
			Location: "header",
			Key:      "Authorization",
		}
		// Check if it mentions realm with oauth hints.
		if strings.Contains(lower, "realm") {
			// Could refine further, but bearer is the primary indicator.
		}
		return info

	case strings.HasPrefix(lower, "basic"):
		return &models.AuthInfo{
			Type:     models.AuthBasic,
			Location: "header",
			Key:      "Authorization",
		}

	case strings.HasPrefix(lower, "negotiate"), strings.HasPrefix(lower, "ntlm"):
		// Not directly supported but report as basic for now.
		return &models.AuthInfo{
			Type:     models.AuthBasic,
			Location: "header",
			Key:      "Authorization",
		}

	default:
		return nil
	}
}

// probeAuthEndpoints checks common auth endpoints to detect OAuth2 or login flows.
func (d *Detector) probeAuthEndpoints(ctx context.Context) (*models.AuthInfo, error) {
	for _, path := range commonAuthEndpoints {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		url := d.baseURL + path
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			continue
		}

		resp, err := d.client.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()

		// A non-404 response on an auth endpoint suggests it exists.
		if resp.StatusCode == http.StatusNotFound {
			continue
		}

		// Detect OAuth2 endpoints.
		if strings.Contains(path, "oauth") || strings.Contains(path, "token") {
			return &models.AuthInfo{
				Type:     models.AuthOAuth2,
				Location: "header",
				Key:      "Authorization",
			}, nil
		}

		// A login endpoint suggests some form of session/cookie auth or basic auth.
		if strings.Contains(path, "login") || strings.Contains(path, "auth") {
			return &models.AuthInfo{
				Type:     models.AuthBearer,
				Location: "header",
				Key:      "Authorization",
			}, nil
		}
	}

	return nil, nil
}
