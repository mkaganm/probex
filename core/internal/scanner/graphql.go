package scanner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/probex/probex/internal/models"
	"github.com/tidwall/gjson"
)

// GraphQLScanner discovers GraphQL schema and operations.
type GraphQLScanner struct {
	baseURL    string
	endpoint   string // typically /graphql
	client     *http.Client
	authHeader string
}

// GraphQLType represents a GraphQL type from introspection.
type GraphQLType struct {
	Name        string          `json:"name"`
	Kind        string          `json:"kind"`       // OBJECT, SCALAR, ENUM, INPUT_OBJECT, LIST, NON_NULL
	Description string          `json:"description"`
	Fields      []GraphQLField  `json:"fields"`
	InputFields []GraphQLField  `json:"inputFields"`
	EnumValues  []GraphQLEnum   `json:"enumValues"`
	OfType      *GraphQLType    `json:"ofType"`
}

// GraphQLField represents a field in a GraphQL type.
type GraphQLField struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Type        GraphQLTypeRef  `json:"type"`
	Args        []GraphQLArg    `json:"args"`
}

// GraphQLArg is a GraphQL field argument.
type GraphQLArg struct {
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Type         GraphQLTypeRef `json:"type"`
	DefaultValue *string        `json:"defaultValue"`
}

// GraphQLTypeRef is a type reference (possibly wrapped in NON_NULL/LIST).
type GraphQLTypeRef struct {
	Kind   string          `json:"kind"`
	Name   *string         `json:"name"`
	OfType *GraphQLTypeRef `json:"ofType"`
}

// GraphQLEnum is an enum value.
type GraphQLEnum struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// GraphQLSchema holds the introspected schema.
type GraphQLSchema struct {
	QueryType        *GraphQLType  `json:"queryType"`
	MutationType     *GraphQLType  `json:"mutationType"`
	SubscriptionType *GraphQLType  `json:"subscriptionType"`
	Types            []GraphQLType `json:"types"`
}

// NewGraphQLScanner creates a new GraphQL scanner.
func NewGraphQLScanner(baseURL string) *GraphQLScanner {
	return &GraphQLScanner{
		baseURL:  strings.TrimRight(baseURL, "/"),
		endpoint: "/graphql",
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// SetEndpoint sets the GraphQL endpoint path (default: /graphql).
func (gs *GraphQLScanner) SetEndpoint(path string) {
	gs.endpoint = path
}

// SetAuth sets the authorization header.
func (gs *GraphQLScanner) SetAuth(header string) {
	gs.authHeader = header
}

// introspectionQuery is the standard GraphQL introspection query.
const introspectionQuery = `{
  __schema {
    queryType { name }
    mutationType { name }
    subscriptionType { name }
    types {
      name
      kind
      description
      fields(includeDeprecated: true) {
        name
        description
        type {
          kind
          name
          ofType { kind name ofType { kind name ofType { kind name } } }
        }
        args {
          name
          description
          type {
            kind
            name
            ofType { kind name ofType { kind name } }
          }
          defaultValue
        }
      }
      inputFields {
        name
        description
        type {
          kind
          name
          ofType { kind name ofType { kind name } }
        }
      }
      enumValues(includeDeprecated: true) {
        name
        description
      }
    }
  }
}`

// Discover runs GraphQL introspection and returns discovered operations as endpoints.
func (gs *GraphQLScanner) Discover(ctx context.Context) ([]models.Endpoint, error) {
	schema, err := gs.introspect(ctx)
	if err != nil {
		return nil, err
	}

	var endpoints []models.Endpoint

	// Build type map for resolving references.
	typeMap := make(map[string]GraphQLType)
	for _, t := range schema.Types {
		typeMap[t.Name] = t
	}

	// Extract queries.
	if schema.QueryType != nil && schema.QueryType.Name != "" {
		if qt, ok := typeMap[schema.QueryType.Name]; ok {
			for _, field := range qt.Fields {
				ep := gs.fieldToEndpoint(field, "query", typeMap)
				endpoints = append(endpoints, ep)
			}
		}
	}

	// Extract mutations.
	if schema.MutationType != nil && schema.MutationType.Name != "" {
		if mt, ok := typeMap[schema.MutationType.Name]; ok {
			for _, field := range mt.Fields {
				ep := gs.fieldToEndpoint(field, "mutation", typeMap)
				endpoints = append(endpoints, ep)
			}
		}
	}

	// Extract subscriptions.
	if schema.SubscriptionType != nil && schema.SubscriptionType.Name != "" {
		if st, ok := typeMap[schema.SubscriptionType.Name]; ok {
			for _, field := range st.Fields {
				ep := gs.fieldToEndpoint(field, "subscription", typeMap)
				endpoints = append(endpoints, ep)
			}
		}
	}

	return endpoints, nil
}

// DetectGraphQL checks if a URL exposes a GraphQL endpoint.
func (gs *GraphQLScanner) DetectGraphQL(ctx context.Context) bool {
	paths := []string{"/graphql", "/api/graphql", "/gql", "/query"}
	for _, path := range paths {
		url := gs.baseURL + path
		body := `{"query":"{ __typename }"}`
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(body))
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
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()

		if resp.StatusCode == 200 && gjson.GetBytes(respBody, "data").Exists() {
			gs.endpoint = path
			return true
		}
	}
	return false
}

func (gs *GraphQLScanner) introspect(ctx context.Context) (*GraphQLSchema, error) {
	reqBody, _ := json.Marshal(map[string]string{"query": introspectionQuery})

	url := gs.baseURL + gs.endpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if gs.authHeader != "" {
		req.Header.Set("Authorization", gs.authHeader)
	}

	resp, err := gs.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("introspection request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("introspection returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return nil, err
	}

	// Parse the introspection result.
	schemaData := gjson.GetBytes(body, "data.__schema")
	if !schemaData.Exists() {
		return nil, fmt.Errorf("no __schema in introspection response")
	}

	var schema GraphQLSchema
	if err := json.Unmarshal([]byte(schemaData.Raw), &schema); err != nil {
		return nil, fmt.Errorf("parse introspection schema: %w", err)
	}

	return &schema, nil
}

func (gs *GraphQLScanner) fieldToEndpoint(field GraphQLField, opType string, typeMap map[string]GraphQLType) models.Endpoint {
	// Map GraphQL operation type to HTTP method convention.
	method := "QUERY"
	if opType == "mutation" {
		method = "MUTATION"
	} else if opType == "subscription" {
		method = "SUBSCRIPTION"
	}

	ep := models.Endpoint{
		ID:           endpointID(method, gs.endpoint+"/"+field.Name),
		Method:       method,
		Path:         gs.endpoint,
		BaseURL:      gs.baseURL,
		Tags:         []string{"graphql", opType},
		DiscoveredAt: time.Now(),
		Source:       models.SourceOpenAPI, // closest match
	}

	// Convert args to query params.
	for _, arg := range field.Args {
		ep.QueryParams = append(ep.QueryParams, models.Parameter{
			Name:     arg.Name,
			Type:     resolveTypeName(arg.Type),
			Required: arg.Type.Kind == "NON_NULL",
		})
	}

	// Build request body schema from args.
	if len(field.Args) > 0 {
		ep.RequestBody = argsToSchema(field.Args, typeMap)
	}

	// Build response schema from return type.
	returnType := resolveTypeName(field.Type)
	ep.Responses = []models.Response{
		{
			StatusCode:  200,
			ContentType: "application/json",
			Schema: &models.Schema{
				Type: "object",
				Properties: map[string]*models.Schema{
					"data": {
						Type: "object",
						Properties: map[string]*models.Schema{
							field.Name: {Type: returnType},
						},
					},
				},
			},
		},
	}

	// Set description.
	ep.Headers = map[string]string{
		"X-GraphQL-Operation": field.Name,
		"X-GraphQL-Type":      opType,
	}

	return ep
}

func argsToSchema(args []GraphQLArg, typeMap map[string]GraphQLType) *models.Schema {
	schema := &models.Schema{
		Type:       "object",
		Properties: make(map[string]*models.Schema),
	}
	for _, arg := range args {
		typeName := resolveTypeName(arg.Type)
		propSchema := graphqlTypeToSchema(typeName, typeMap)
		schema.Properties[arg.Name] = propSchema
		if arg.Type.Kind == "NON_NULL" {
			schema.Required = append(schema.Required, arg.Name)
		}
	}
	return schema
}

func graphqlTypeToSchema(typeName string, typeMap map[string]GraphQLType) *models.Schema {
	switch typeName {
	case "String", "ID":
		return &models.Schema{Type: "string"}
	case "Int":
		return &models.Schema{Type: "integer"}
	case "Float":
		return &models.Schema{Type: "number"}
	case "Boolean":
		return &models.Schema{Type: "boolean"}
	default:
		// Check if it's an enum or input object.
		if t, ok := typeMap[typeName]; ok {
			switch t.Kind {
			case "ENUM":
				vals := make([]any, len(t.EnumValues))
				for i, v := range t.EnumValues {
					vals[i] = v.Name
				}
				return &models.Schema{Type: "string", Enum: vals}
			case "INPUT_OBJECT":
				s := &models.Schema{Type: "object", Properties: make(map[string]*models.Schema)}
				for _, f := range t.InputFields {
					s.Properties[f.Name] = graphqlTypeToSchema(resolveTypeName(f.Type), typeMap)
				}
				return s
			}
		}
		return &models.Schema{Type: "object"}
	}
}

// resolveTypeName unwraps NON_NULL and LIST wrappers to get the base type name.
func resolveTypeName(ref GraphQLTypeRef) string {
	if ref.Name != nil {
		return *ref.Name
	}
	if ref.OfType != nil {
		return resolveTypeName(*ref.OfType)
	}
	return "String"
}
