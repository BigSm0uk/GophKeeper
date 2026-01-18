package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/client/api"
	"github.com/BigSm0uk/GophKeeper/internal/client/storage"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type AuthMode int

const (
	ModeLogin AuthMode = iota
	ModeRegister
)

type authModel struct {
	mode       AuthMode
	username   textinput.Model
	password   textinput.Model
	email      textinput.Model
	focused    int
	loading    bool
	err        error
	success    string
	client     *api.Client
	tokenStore *storage.TokenStore
	spinner    spinner.Model
}

func NewAuthModel(client *api.Client, store *storage.TokenStore, mode AuthMode) authModel {
	u := textinput.New()
	u.Placeholder = "Username"
	u.CharLimit = 50
	u.Focus()

	p := textinput.New()
	p.Placeholder = "Password"
	p.CharLimit = 128
	p.EchoMode = textinput.EchoPassword

	e := textinput.New()
	e.Placeholder = "Email (optional)"
	e.CharLimit = 100

	s := spinner.New()
	s.Spinner = spinner.Dot

	return authModel{
		mode:       mode,
		username:   u,
		password:   p,
		email:      e,
		focused:    0,
		client:     client,
		tokenStore: store,
		spinner:    s,
	}
}

func (m authModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m authModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "tab", "shift+tab", "up", "down":
			m.focused = (m.focused + 1) % m.fieldsCount()
			return m, m.focus()
		case "enter":
			if m.loading {
				return m, nil
			}
			return m.startSubmit()
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case submitResult:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.success = msg.message
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	switch m.focused {
	case 0:
		m.username, cmd = m.username.Update(msg)
	case 1:
		m.password, cmd = m.password.Update(msg)
	case 2:
		if m.mode == ModeRegister {
			m.email, cmd = m.email.Update(msg)
		}
	}
	return m, cmd
}

func (m authModel) View() string {
	title := "Login"
	if m.mode == ModeRegister {
		title = "Register"
	}

	form := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render("GophKeeper "+title),
		m.username.View(),
		m.password.View(),
	)
	if m.mode == ModeRegister {
		form = lipgloss.JoinVertical(lipgloss.Left, form, m.email.View())
	}

	status := ""
	if m.loading {
		status = m.spinner.View() + " Processing..."
	} else if m.err != nil {
		status = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("Error: " + m.err.Error())
	} else if m.success != "" {
		status = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render(m.success)
	}

	help := "Enter: submit | Tab: next | Esc/Ctrl+C: exit"

	return lipgloss.JoinVertical(lipgloss.Left,
		form,
		"",
		status,
		"",
		help,
	)
}

func (m *authModel) fieldsCount() int {
	if m.mode == ModeRegister {
		return 3
	}
	return 2
}

func (m authModel) focus() tea.Cmd {
	switch m.focused {
	case 0:
		cmd := m.username.Focus()
		m.password.Blur()
		m.email.Blur()
		return cmd
	case 1:
		m.username.Blur()
		cmd := m.password.Focus()
		m.email.Blur()
		return cmd
	case 2:
		m.username.Blur()
		m.password.Blur()
		cmd := m.email.Focus()
		return cmd
	}
	return nil
}

type submitResult struct {
	message string
	err     error
}

func (m authModel) startSubmit() (tea.Model, tea.Cmd) {
	m.loading = true
	m.err = nil
	username := m.username.Value()
	password := m.password.Value()
	email := m.email.Value()

	return m, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		if m.mode == ModeRegister {
			if _, err := m.client.Register(ctx, username, password, email); err != nil {
				return submitResult{err: err}
			}
		}

		tokens, err := m.client.PasswordToken(ctx, username, password, "gophkeeper-cli", "", "offline_access")
		if err != nil {
			return submitResult{err: err}
		}

		if err := m.tokenStore.SaveAccessToken(username, tokens.AccessToken); err != nil {
			return submitResult{err: fmt.Errorf("save access token: %w", err)}
		}
		if err := m.tokenStore.SaveRefreshToken(username, tokens.RefreshToken); err != nil {
			return submitResult{err: fmt.Errorf("save refresh token: %w", err)}
		}
		m.client.SetAccessToken(tokens.AccessToken)

		action := "Logged in"
		if m.mode == ModeRegister {
			action = "Registered and logged in"
		}
		return submitResult{message: action}
	}
}
