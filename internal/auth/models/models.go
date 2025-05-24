package models

// UserInfo represents authenticated user information from any provider
type UserInfo struct {
	ID       string
	Email    string
	Name     string
	Picture  string
	Metadata map[string]interface{}
}

// AuthorizationCode represents a stored authorization code
type AuthorizationCode struct {
	Code                string
	CodeChallenge       string
	CodeChallengeMethod string
	UserID              string
	ExpiresAt           int64
}
