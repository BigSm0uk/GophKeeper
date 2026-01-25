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
	ModeSelect AuthMode = iota
	ModeLogin
	ModeRegister
)

type authModel struct {
	mode        AuthMode
	username    textinput.Model
	password    textinput.Model
	email       textinput.Model
	focused     int
	loading     bool
	err         error
	success     string
	authUser    string
	authAccess  string
	authRefresh string
	client      *api.Client
	tokenStore  *storage.TokenStore
	spinner     spinner.Model
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("62")).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true).
			Padding(0, 1)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)

	buttonStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62"))

	selectedButtonStyle = buttonStyle.Copy().
				Foreground(lipgloss.Color("62")).
				BorderForeground(lipgloss.Color("62"))
)

func NewAuthModel(client *api.Client, store *storage.TokenStore, mode AuthMode) authModel {
	u := textinput.New()
	u.Placeholder = "Username"
	u.CharLimit = 50
	u.Width = 40
	u.Prompt = "> "
	u.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("62"))
	u.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	u.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("62"))

	p := textinput.New()
	p.Placeholder = "Password"
	p.CharLimit = 128
	p.Width = 40
	p.Prompt = "> "
	p.EchoMode = textinput.EchoPassword
	p.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("62"))
	p.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	p.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("62"))

	e := textinput.New()
	e.Placeholder = "Email (optional)"
	e.CharLimit = 100
	e.Width = 40
	e.Prompt = "> "
	e.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("62"))
	e.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	e.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("62"))

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("62"))

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
	if m.mode == ModeSelect {
		return nil
	}
	// Возвращаем команду фокусировки для первого поля
	return m.username.Focus()
}

func (m authModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Обрабатываем специальные сообщения
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Передаем размер окна в поля ввода для правильного отображения
		return m, nil
	case tea.KeyMsg:
		key := msg.String()

		// Специальные клавиши для всех режимов
		if key == "ctrl+c" || key == "q" {
			return m, tea.Quit
		}

		if key == "esc" {
			if m.mode != ModeSelect {
				m.mode = ModeSelect
				m.err = nil
				m.success = ""
				m.username.SetValue("")
				m.password.SetValue("")
				m.email.SetValue("")
				m.username.Blur()
				m.password.Blur()
				m.email.Blur()
				return m, nil
			}
			return m, tea.Quit
		}

		// Обработка для режима выбора
		if m.mode == ModeSelect {
			switch key {
			case "tab", "down":
				m.focused = (m.focused + 1) % 2
				return m, nil
			case "shift+tab", "up":
				m.focused = (m.focused - 1 + 2) % 2
				return m, nil
			case "enter":
				if m.focused == 0 {
					m.mode = ModeLogin
					m.focused = 0
					m.err = nil
					m.success = ""
					m.password.Blur()
					m.email.Blur()
					cmd := m.username.Focus()
					return m, cmd
				} else if m.focused == 1 {
					m.mode = ModeRegister
					m.focused = 0
					m.err = nil
					m.success = ""
					m.password.Blur()
					m.email.Blur()
					cmd := m.username.Focus()
					return m, cmd
				}
				return m, nil
			case "1":
				m.mode = ModeLogin
				m.focused = 0
				m.err = nil
				m.success = ""
				m.password.Blur()
				m.email.Blur()
				cmd := m.username.Focus()
				return m, cmd
			case "2":
				m.mode = ModeRegister
				m.focused = 0
				m.err = nil
				m.success = ""
				m.password.Blur()
				m.email.Blur()
				cmd := m.username.Focus()
				return m, cmd
			}
			return m, nil
		}

		// Обработка для режима ввода (Login/Register)
		if m.loading {
			return m, nil
		}

		// Обрабатываем специальные клавиши для навигации
		switch key {
		case "tab":
			m.focused = (m.focused + 1) % m.fieldsCount()
			// Устанавливаем фокус на нужное поле
			switch m.focused {
			case 0:
				m.password.Blur()
				m.email.Blur()
				cmd := m.username.Focus()
				return m, cmd
			case 1:
				m.username.Blur()
				m.email.Blur()
				cmd := m.password.Focus()
				return m, cmd
			case 2:
				m.username.Blur()
				m.password.Blur()
				cmd := m.email.Focus()
				return m, cmd
			}
			return m, nil
		case "shift+tab", "up":
			m.focused = (m.focused - 1 + m.fieldsCount()) % m.fieldsCount()
			// Устанавливаем фокус на нужное поле
			switch m.focused {
			case 0:
				m.password.Blur()
				m.email.Blur()
				cmd := m.username.Focus()
				return m, cmd
			case 1:
				m.username.Blur()
				m.email.Blur()
				cmd := m.password.Focus()
				return m, cmd
			case 2:
				m.username.Blur()
				m.password.Blur()
				cmd := m.email.Focus()
				return m, cmd
			}
			return m, nil
		case "down":
			m.focused = (m.focused + 1) % m.fieldsCount()
			// Устанавливаем фокус на нужное поле
			switch m.focused {
			case 0:
				m.password.Blur()
				m.email.Blur()
				cmd := m.username.Focus()
				return m, cmd
			case 1:
				m.username.Blur()
				m.email.Blur()
				cmd := m.password.Focus()
				return m, cmd
			case 2:
				m.username.Blur()
				m.password.Blur()
				cmd := m.email.Focus()
				return m, cmd
			}
			return m, nil
		case "enter":
			newModel, cmd := m.startSubmit()
			return newModel, cmd
		}

		// ВСЕ остальные клавиши (включая обычные символы) передаем в активное поле ввода
		var cmd tea.Cmd
		switch m.focused {
		case 0:
			m.username, cmd = m.username.Update(msg)
			return m, cmd
		case 1:
			m.password, cmd = m.password.Update(msg)
			return m, cmd
		case 2:
			if m.mode == ModeRegister {
				m.email, cmd = m.email.Update(msg)
				return m, cmd
			}
		}
		return m, nil

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
			m.authUser = msg.username
			m.authAccess = msg.accessToken
			m.authRefresh = msg.refreshToken
			// После успешной авторизации ждем немного и завершаем программу
			// Главное меню запустится в auth.go после проверки токена
			return m, tea.Sequence(
				tea.Tick(2*time.Second, func(time.Time) tea.Msg {
					return tea.Quit()
				}),
			)
		}
	}

	// Для всех остальных сообщений (не KeyMsg) передаем в активное поле
	if m.mode != ModeSelect && !m.loading {
		var cmd tea.Cmd
		switch m.focused {
		case 0:
			m.username, cmd = m.username.Update(msg)
			return m, cmd
		case 1:
			m.password, cmd = m.password.Update(msg)
			return m, cmd
		case 2:
			if m.mode == ModeRegister {
				m.email, cmd = m.email.Update(msg)
				return m, cmd
			}
		}
	}

	return m, nil
}

func (m authModel) View() string {
	if m.mode == ModeSelect {
		return m.viewSelect()
	}

	title := "🔐 Login"
	if m.mode == ModeRegister {
		title = "📝 Register"
	}

	// Стили для активного и неактивного поля
	activeFieldStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true)
	inactiveFieldStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	var form string
	if m.mode == ModeRegister {
		usernameLabel := "Username:"
		passwordLabel := "Password:"
		emailLabel := "Email (optional):"

		if m.focused == 0 {
			usernameLabel = "→ " + activeFieldStyle.Render("Username:")
		} else {
			usernameLabel = "  " + inactiveFieldStyle.Render("Username:")
		}
		if m.focused == 1 {
			passwordLabel = "→ " + activeFieldStyle.Render("Password:")
		} else {
			passwordLabel = "  " + inactiveFieldStyle.Render("Password:")
		}
		if m.focused == 2 {
			emailLabel = "→ " + activeFieldStyle.Render("Email (optional):")
		} else {
			emailLabel = "  " + inactiveFieldStyle.Render("Email (optional):")
		}

		form = lipgloss.JoinVertical(lipgloss.Left,
			usernameLabel,
			"  "+m.username.View(),
			"",
			passwordLabel,
			"  "+m.password.View(),
			"",
			emailLabel,
			"  "+m.email.View(),
		)
	} else {
		usernameLabel := "Username:"
		passwordLabel := "Password:"

		if m.focused == 0 {
			usernameLabel = "→ " + activeFieldStyle.Render("Username:")
		} else {
			usernameLabel = "  " + inactiveFieldStyle.Render("Username:")
		}
		if m.focused == 1 {
			passwordLabel = "→ " + activeFieldStyle.Render("Password:")
		} else {
			passwordLabel = "  " + inactiveFieldStyle.Render("Password:")
		}

		form = lipgloss.JoinVertical(lipgloss.Left,
			usernameLabel,
			"  "+m.username.View(),
			"",
			passwordLabel,
			"  "+m.password.View(),
		)
	}

	status := ""
	if m.loading {
		status = "\n  " + m.spinner.View() + " Processing..."
	} else if m.err != nil {
		status = "\n  " + errorStyle.Render("❌ Error: "+m.err.Error())
	} else if m.success != "" {
		status = "\n  " + successStyle.Render("✅ "+m.success)
	}

	help := helpStyle.Render("\n  Enter: submit | Tab: next field | Esc: back | Ctrl+C: exit")

	content := lipgloss.JoinVertical(lipgloss.Center,
		titleStyle.Render(title),
		"",
		form,
		status,
		help,
	)

	return lipgloss.Place(
		lipgloss.Width(content),
		lipgloss.Height(content),
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

func (m authModel) viewSelect() string {
	loginBtn := buttonStyle.Render("1. Login")
	registerBtn := buttonStyle.Render("2. Register")

	activeIndicator := lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Render("→")
	inactiveIndicator := "  "

	if m.focused == 0 {
		loginBtn = selectedButtonStyle.Render("1. Login")
	} else if m.focused == 1 {
		registerBtn = selectedButtonStyle.Render("2. Register")
	}

	loginLine := inactiveIndicator + loginBtn
	registerLine := inactiveIndicator + registerBtn

	if m.focused == 0 {
		loginLine = activeIndicator + loginBtn
	} else if m.focused == 1 {
		registerLine = activeIndicator + registerBtn
	}

	// Создаем контент с выровненными кнопками
	buttons := lipgloss.JoinVertical(lipgloss.Center,
		loginLine,
		registerLine,
	)

	// Центрируем кнопки
	centeredButtons := lipgloss.PlaceHorizontal(60, lipgloss.Center, buttons)

	content := lipgloss.JoinVertical(lipgloss.Center,
		titleStyle.Render("🔐 GophKeeper"),
		"",
		"Welcome! Please choose an option:",
		"",
		centeredButtons,
		"",
		helpStyle.Render("Enter/1: Login | Enter/2: Register | Ctrl+C: exit"),
	)

	// Центрируем весь контент на экране
	return lipgloss.Place(
		80,
		24,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

func (m *authModel) fieldsCount() int {
	if m.mode == ModeRegister {
		return 3
	}
	return 2
}

func (m *authModel) focus() tea.Cmd {
	switch m.focused {
	case 0:
		m.password.Blur()
		m.email.Blur()
		cmd := m.username.Focus()
		return cmd
	case 1:
		m.username.Blur()
		m.email.Blur()
		cmd := m.password.Focus()
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
	message      string
	username     string
	accessToken  string
	refreshToken string
	err          error
}

func (m authModel) startSubmit() (tea.Model, tea.Cmd) {
	m.loading = true
	m.err = nil
	username := m.username.Value()
	password := m.password.Value()
	email := m.email.Value()

	if username == "" {
		m.loading = false
		m.err = fmt.Errorf("username is required")
		return m, nil
	}
	if password == "" {
		m.loading = false
		m.err = fmt.Errorf("password is required")
		return m, nil
	}

	return m, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		if m.mode == ModeRegister {
			resp, err := m.client.Register(ctx, username, password, email)
			if err != nil {
				return submitResult{err: fmt.Errorf("registration failed: %w", err)}
			}
			_ = resp // используем для проверки успешности
		}

		tokens, err := m.client.PasswordToken(ctx, username, password, "gophkeeper-cli", "", "offline_access")
		if err != nil {
			return submitResult{err: fmt.Errorf("authentication failed: %w", err)}
		}

		if err := m.tokenStore.SaveAccessToken(username, tokens.AccessToken); err != nil {
			return submitResult{err: fmt.Errorf("failed to save access token: %w", err)}
		}
		if err := m.tokenStore.SaveRefreshToken(username, tokens.RefreshToken); err != nil {
			return submitResult{err: fmt.Errorf("failed to save refresh token: %w", err)}
		}
		// Сохраняем текущий username для последующего использования
		if err := m.tokenStore.SaveCurrentUsername(username); err != nil {
			return submitResult{err: fmt.Errorf("failed to save current username: %w", err)}
		}
		m.client.SetAccessToken(tokens.AccessToken)

		action := "Successfully logged in!"
		if m.mode == ModeRegister {
			action = "Successfully registered and logged in!"
		}
		return submitResult{
			message:      action,
			username:     username,
			accessToken:  tokens.AccessToken,
			refreshToken: tokens.RefreshToken,
		}
	}
}
