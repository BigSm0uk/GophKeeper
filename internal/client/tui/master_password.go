package tui

import (
	"fmt"

	"github.com/BigSm0uk/GophKeeper/internal/client/storage"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MasterPasswordModel представляет экран ввода мастер-пароля для шифрования.
type MasterPasswordModel struct {
	username      string
	password      textinput.Model
	confirmPass   textinput.Model
	focused       int
	err           error
	success       bool
	isNewUser     bool
	tokenStore    *storage.TokenStore
	dbPath        string
	storageResult *storage.StorageManager
}

// NewMasterPasswordModel creates a new master password input screen.
func NewMasterPasswordModel(username, dbPath string, tokenStore *storage.TokenStore) MasterPasswordModel {
	p := textinput.New()
	p.Placeholder = "Master Password"
	p.CharLimit = 128
	p.Width = 40
	p.Prompt = "> "
	p.EchoMode = textinput.EchoPassword
	p.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("62"))
	p.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	p.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("62"))

	cp := textinput.New()
	cp.Placeholder = "Confirm Password"
	cp.CharLimit = 128
	cp.Width = 40
	cp.Prompt = "> "
	cp.EchoMode = textinput.EchoPassword
	cp.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("62"))
	cp.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	cp.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("62"))

	// Проверяем, существует ли salt для этого пользователя
	salt, _ := tokenStore.GetEncryptionSalt(username)
	isNewUser := salt == nil || len(salt) == 0

	// Устанавливаем фокус на первое поле сразу
	p.Focus()

	return MasterPasswordModel{
		username:    username,
		password:    p,
		confirmPass: cp,
		focused:     0,
		isNewUser:   isNewUser,
		tokenStore:  tokenStore,
		dbPath:      dbPath,
	}
}

func (m MasterPasswordModel) Init() tea.Cmd {
	return m.password.Focus()
}

func (m MasterPasswordModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		if key == "ctrl+c" || key == "esc" {
			return m, tea.Quit
		}

		// Обработка Enter
		if key == "enter" {
			if m.isNewUser {
				// Для нового пользователя нужно подтверждение
				if m.focused == 0 {
					// Переход на поле подтверждения
					m.focused = 1
					m.password.Blur()
					return m, m.confirmPass.Focus()
				} else {
					// Проверка совпадения паролей
					if m.password.Value() != m.confirmPass.Value() {
						m.err = fmt.Errorf("passwords do not match")
						return m, nil
					}
					return m, m.initStorage()
				}
			} else {
				// Для существующего пользователя сразу инициализация
				return m, m.initStorage()
			}
		}

		// Навигация Tab
		if key == "tab" && m.isNewUser {
			m.focused = (m.focused + 1) % 2
			if m.focused == 0 {
				m.confirmPass.Blur()
				return m, m.password.Focus()
			} else {
				m.password.Blur()
				return m, m.confirmPass.Focus()
			}
		}

	case storageInitResult:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.success = true
		m.storageResult = msg.storage
		// Завершаем с небольшой задержкой
		return m, tea.Quit
	}

	// Обновление активного поля
	var cmd tea.Cmd
	if m.focused == 0 {
		m.password, cmd = m.password.Update(msg)
	} else if m.isNewUser {
		m.confirmPass, cmd = m.confirmPass.Update(msg)
	}

	return m, cmd
}

func (m MasterPasswordModel) View() string {
	title := "🔐 Master Password"
	if m.isNewUser {
		title = "🔐 Create Master Password"
	}

	// Стили
	activeFieldStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true)
	inactiveFieldStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	var form string
	if m.isNewUser {
		passwordLabel := "Master Password:"
		confirmLabel := "Confirm Password:"

		if m.focused == 0 {
			passwordLabel = "→ " + activeFieldStyle.Render("Master Password:")
		} else {
			passwordLabel = "  " + inactiveFieldStyle.Render("Master Password:")
		}

		if m.focused == 1 {
			confirmLabel = "→ " + activeFieldStyle.Render("Confirm Password:")
		} else {
			confirmLabel = "  " + inactiveFieldStyle.Render("Confirm Password:")
		}

		form = lipgloss.JoinVertical(lipgloss.Left,
			passwordLabel,
			"  "+m.password.View(),
			"",
			confirmLabel,
			"  "+m.confirmPass.View(),
		)
	} else {
		passwordLabel := "→ " + activeFieldStyle.Render("Master Password:")
		form = lipgloss.JoinVertical(lipgloss.Left,
			passwordLabel,
			"  "+m.password.View(),
		)
	}

	// Информационное сообщение
	var info string
	if m.isNewUser {
		info = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true).
			Render("\n  This password will encrypt all your data locally.")
	} else {
		info = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true).
			Render("\n  Enter your master password to unlock local storage.")
	}

	// Статус
	status := ""
	if m.err != nil {
		status = "\n  " + errorStyle.Render("❌ Error: "+m.err.Error())
	}

	// Подсказка
	help := ""
	if m.isNewUser {
		help = helpStyle.Render("\n  Enter: next/submit | Tab: switch field | Esc: exit")
	} else {
		help = helpStyle.Render("\n  Enter: unlock | Esc: exit")
	}

	content := lipgloss.JoinVertical(lipgloss.Center,
		titleStyle.Render(title),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("User: "+m.username),
		info,
		"",
		form,
		status,
		help,
	)

	return lipgloss.Place(
		80,
		24,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

type storageInitResult struct {
	storage *storage.StorageManager
	err     error
}

func (m MasterPasswordModel) initStorage() tea.Cmd {
	return func() tea.Msg {
		masterPassword := m.password.Value()

		if masterPassword == "" {
			return storageInitResult{err: fmt.Errorf("master password is required")}
		}

		// Инициализация StorageManager
		sm, err := storage.InitializeStorage(m.dbPath, m.username, masterPassword)
		if err != nil {
			return storageInitResult{err: fmt.Errorf("failed to initialize storage: %w", err)}
		}

		return storageInitResult{storage: sm}
	}
}

// GetStorageManager returns the initialized storage manager if successful.
func (m MasterPasswordModel) GetStorageManager() *storage.StorageManager {
	return m.storageResult
}

// IsSuccess returns true if master password was successfully entered.
func (m MasterPasswordModel) IsSuccess() bool {
	return m.success
}
