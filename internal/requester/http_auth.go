package requester

import (
	"fmt"
	"net/http"

	"github.com/brizzai/auto-mcp/internal/config"
)

// AuthManager handles request authentication
type AuthManager interface {
	ApplyAuth(req *http.Request) error
}

// HTTPAuthManager implements the AuthManager interface
type HTTPAuthManager struct {
	authType   config.AuthType
	authConfig map[string]string
}

// NewHTTPAuthManager creates a new HTTPAuthManager
func NewHTTPAuthManager(serviceConfig *config.EndpointConfig) *HTTPAuthManager {
	return &HTTPAuthManager{
		authType:   serviceConfig.AuthType,
		authConfig: serviceConfig.AuthConfig,
	}
}

// ApplyAuth adds authentication to the request
func (a *HTTPAuthManager) ApplyAuth(req *http.Request) error {
	switch a.authType {
	case config.AuthTypeNone:
		return nil
	case config.AuthTypeBasic:
		username := a.authConfig["username"]
		password := a.authConfig["password"]
		req.SetBasicAuth(username, password)
	case config.AuthTypeBearer:
		token := a.authConfig["token"]
		req.Header.Set("Authorization", "Bearer "+token)
	case config.AuthTypeAPIKey:
		key := a.authConfig["key"]
		header := a.authConfig["header"]
		if header == "" {
			header = "X-API-Key"
		}
		req.Header.Set(header, key)
	case config.AuthTypeOAuth2:
		token := a.authConfig["token"]
		req.Header.Set("Authorization", "Bearer "+token)
	default:
		return fmt.Errorf("unsupported auth type: %s", a.authType)
	}
	return nil
}
