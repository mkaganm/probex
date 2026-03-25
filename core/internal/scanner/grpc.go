package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mkaganm/probex/internal/models"
	"github.com/tidwall/gjson"
)

// GRPCScanner discovers gRPC services via server reflection or known patterns.
type GRPCScanner struct {
	baseURL    string
	client     *http.Client
	authHeader string
}

// GRPCService represents a discovered gRPC service.
type GRPCService struct {
	Name    string       `json:"name"`
	Methods []GRPCMethod `json:"methods"`
}

// GRPCMethod represents a gRPC method.
type GRPCMethod struct {
	Name            string `json:"name"`
	FullName        string `json:"full_name"`
	ClientStreaming bool   `json:"client_streaming"`
	ServerStreaming bool   `json:"server_streaming"`
	InputType       string `json:"input_type"`
	OutputType      string `json:"output_type"`
}

// NewGRPCScanner creates a new gRPC scanner.
func NewGRPCScanner(baseURL string) *GRPCScanner {
	return &GRPCScanner{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

// SetAuth sets the authorization header.
func (gs *GRPCScanner) SetAuth(header string) {
	gs.authHeader = header
}

// DetectGRPC checks if the target supports gRPC or gRPC-Web.
func (gs *GRPCScanner) DetectGRPC(ctx context.Context) bool {
	// Check for gRPC-Web by looking at common paths.
	paths := []string{"/grpc", "/api/grpc"}
	for _, path := range paths {
		url := gs.baseURL + path
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(""))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/grpc-web+proto")
		if gs.authHeader != "" {
			req.Header.Set("Authorization", gs.authHeader)
		}

		resp, err := gs.client.Do(req)
		if err != nil {
			continue
		}

		// gRPC services often return specific headers or 415 for wrong content type.
		ct := resp.Header.Get("Content-Type")
		statusCode := resp.StatusCode
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if strings.Contains(ct, "grpc") || statusCode == 415 {
			return true
		}
	}

	// Check for gRPC health check endpoint (gRPC HTTP/JSON transcoding).
	healthPaths := []string{
		"/grpc.health.v1.Health/Check",
		"/api/health",
	}
	for _, path := range healthPaths {
		url := gs.baseURL + path
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url,
			strings.NewReader(`{}`))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		if gs.authHeader != "" {
			req.Header.Set("Authorization", gs.authHeader)
		}

		resp, err := gs.client.Do(req)
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		statusCode := resp.StatusCode
		resp.Body.Close()

		if statusCode == 200 && gjson.ValidBytes(body) {
			status := gjson.GetBytes(body, "status")
			if status.Exists() {
				return true
			}
		}
	}

	return false
}

// DiscoverViaReflection attempts to discover services using gRPC server reflection
// through a gRPC-Web or JSON transcoding proxy.
func (gs *GRPCScanner) DiscoverViaReflection(ctx context.Context) ([]GRPCService, error) {
	// Try the gRPC reflection API via JSON transcoding.
	reflectURL := gs.baseURL + "/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo"
	reqBody := `{"list_services":""}`

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reflectURL, strings.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if gs.authHeader != "" {
		req.Header.Set("Authorization", gs.authHeader)
	}

	resp, err := gs.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("reflection request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("reflection returned status %d", resp.StatusCode)
	}

	// Parse reflection response.
	services := gjson.GetBytes(body, "listServicesResponse.service")
	if !services.Exists() {
		return nil, fmt.Errorf("no services in reflection response")
	}

	var result []GRPCService
	services.ForEach(func(_, value gjson.Result) bool {
		name := value.Get("name").String()
		if name == "" {
			return true
		}
		// Skip internal services.
		if strings.HasPrefix(name, "grpc.reflection") || strings.HasPrefix(name, "grpc.health") {
			return true
		}
		result = append(result, GRPCService{Name: name})
		return true
	})

	return result, nil
}

// DiscoverCommonServices checks for well-known gRPC service patterns.
func (gs *GRPCScanner) DiscoverCommonServices(ctx context.Context) []GRPCService {
	commonServices := []struct {
		name    string
		methods []string
	}{
		{"UserService", []string{"GetUser", "ListUsers", "CreateUser", "UpdateUser", "DeleteUser"}},
		{"AuthService", []string{"Login", "Logout", "RefreshToken", "ValidateToken"}},
		{"ProductService", []string{"GetProduct", "ListProducts", "CreateProduct", "UpdateProduct"}},
		{"OrderService", []string{"GetOrder", "ListOrders", "CreateOrder", "CancelOrder"}},
		{"NotificationService", []string{"SendNotification", "ListNotifications"}},
	}

	var found []GRPCService
	for _, svc := range commonServices {
		for _, method := range svc.methods {
			path := fmt.Sprintf("/%s/%s", svc.name, method)
			if gs.probeGRPCMethod(ctx, path) {
				// Found at least one method — add the service.
				service := GRPCService{Name: svc.name}
				for _, m := range svc.methods {
					service.Methods = append(service.Methods, GRPCMethod{
						Name:     m,
						FullName: fmt.Sprintf("/%s/%s", svc.name, m),
					})
				}
				found = append(found, service)
				break
			}
		}
	}

	return found
}

// Discover returns endpoints for discovered gRPC services.
func (gs *GRPCScanner) Discover(ctx context.Context) []models.Endpoint {
	var services []GRPCService

	// Try reflection first.
	reflected, err := gs.DiscoverViaReflection(ctx)
	if err == nil && len(reflected) > 0 {
		services = reflected
	} else {
		// Fall back to common service probing.
		services = gs.DiscoverCommonServices(ctx)
	}

	var endpoints []models.Endpoint
	for _, svc := range services {
		for _, method := range svc.Methods {
			ep := gs.methodToEndpoint(svc.Name, method)
			endpoints = append(endpoints, ep)
		}
		// If no methods known (from reflection), create a service-level endpoint.
		if len(svc.Methods) == 0 {
			ep := models.Endpoint{
				ID:           endpointID("GRPC", "/"+svc.Name),
				Method:       "GRPC",
				Path:         "/" + svc.Name,
				BaseURL:      gs.baseURL,
				Tags:         []string{"grpc", "service"},
				DiscoveredAt: time.Now(),
				Source:       models.SourceOpenAPI,
				Headers: map[string]string{
					"X-GRPC-Service": svc.Name,
				},
			}
			endpoints = append(endpoints, ep)
		}
	}

	return endpoints
}

func (gs *GRPCScanner) methodToEndpoint(serviceName string, method GRPCMethod) models.Endpoint {
	fullPath := method.FullName
	if fullPath == "" {
		fullPath = fmt.Sprintf("/%s/%s", serviceName, method.Name)
	}

	streamType := "unary"
	if method.ClientStreaming && method.ServerStreaming {
		streamType = "bidi-stream"
	} else if method.ServerStreaming {
		streamType = "server-stream"
	} else if method.ClientStreaming {
		streamType = "client-stream"
	}

	ep := models.Endpoint{
		ID:           endpointID("GRPC", fullPath),
		Method:       "GRPC",
		Path:         fullPath,
		BaseURL:      gs.baseURL,
		Tags:         []string{"grpc", streamType},
		DiscoveredAt: time.Now(),
		Source:       models.SourceOpenAPI,
		Headers: map[string]string{
			"X-GRPC-Service":    serviceName,
			"X-GRPC-Method":     method.Name,
			"X-GRPC-StreamType": streamType,
		},
		Responses: []models.Response{
			{
				StatusCode:  200,
				ContentType: "application/grpc",
			},
		},
	}

	return ep
}

func (gs *GRPCScanner) probeGRPCMethod(ctx context.Context, path string) bool {
	url := gs.baseURL + path

	// Try JSON transcoding.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(`{}`))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	if gs.authHeader != "" {
		req.Header.Set("Authorization", gs.authHeader)
	}

	resp, err := gs.client.Do(req)
	if err != nil {
		return false
	}
	statusCode := resp.StatusCode
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// A gRPC endpoint typically returns 200, 401, 403, or specific gRPC errors.
	// 404 means it doesn't exist.
	return statusCode != 404 && statusCode != 405
}

// ExportServiceDescriptor returns a JSON description of discovered services.
func (gs *GRPCScanner) ExportServiceDescriptor(services []GRPCService) ([]byte, error) {
	return json.MarshalIndent(services, "", "  ")
}
