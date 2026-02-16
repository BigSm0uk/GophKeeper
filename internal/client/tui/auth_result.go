package tui

import tea "github.com/charmbracelet/bubbletea"

// AuthResult carries successful auth data from the TUI to the caller.
type AuthResult struct {
	Username     string
	AccessToken  string
	RefreshToken string
	Success      bool
}

// ExtractAuthResult returns auth data from a finished auth model, if present.
func ExtractAuthResult(model tea.Model) (AuthResult, bool) {
	switch m := model.(type) {
	case authModel:
		return AuthResult{
			Username:     m.authUser,
			AccessToken:  m.authAccess,
			RefreshToken: m.authRefresh,
			Success:      m.authUser != "" && m.authAccess != "",
		}, true
	case *authModel:
		return AuthResult{
			Username:     m.authUser,
			AccessToken:  m.authAccess,
			RefreshToken: m.authRefresh,
			Success:      m.authUser != "" && m.authAccess != "",
		}, true
	default:
		return AuthResult{}, false
	}
}
