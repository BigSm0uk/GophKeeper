package tui

import (
	"context"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/client/api"
	"github.com/BigSm0uk/GophKeeper/internal/client/service"
	"github.com/BigSm0uk/GophKeeper/internal/client/storage"
	"github.com/BigSm0uk/GophKeeper/internal/client/sync"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type menuItem struct {
	title       string
	description string
	id          string
}

func (i menuItem) FilterValue() string { return i.title }
func (i menuItem) Title() string       { return i.title }
func (i menuItem) Description() string { return i.description }

type mainMenuModel struct {
	list           list.Model
	client         *api.Client
	tokenStore     *storage.TokenStore
	offlineService *service.OfflineService
	storageManager *storage.StorageManager
	syncManager    *sync.Manager
	quitting       bool
	isOnline       bool
	lastCheck      time.Time
}

type (
	logoutMsg             struct{}
	subProgramReturnedMsg struct{}
	onlineStatusMsg       struct{ online bool }
)

var (
	menuTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("62")).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62"))

	menuHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)
)

func NewMainMenuModel(client *api.Client, tokenStore *storage.TokenStore, offlineService *service.OfflineService, storageManager *storage.StorageManager, syncManager *sync.Manager) mainMenuModel {
	items := []list.Item{
		menuItem{
			title:       "🔐 Credentials",
			description: "Manage login/password entries",
			id:          "credentials",
		},
		menuItem{
			title:       "📝 Texts",
			description: "Manage text notes",
			id:          "texts",
		},
		menuItem{
			title:       "💳 Cards",
			description: "Manage bank card information",
			id:          "cards",
		},
		menuItem{
			title:       "📁 Binaries",
			description: "Manage binary files",
			id:          "binaries",
		},
		menuItem{
			title:       "🔄 Sync",
			description: "Synchronize data with server",
			id:          "sync",
		},
		menuItem{
			title:       "🚪 Logout",
			description: "Log out and exit",
			id:          "logout",
		},
	}

	l := list.New(items, list.NewDefaultDelegate(), 50, 20)
	l.Title = "GophKeeper - Main Menu"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = menuTitleStyle

	return mainMenuModel{
		list:           l,
		client:         client,
		tokenStore:     tokenStore,
		offlineService: offlineService,
		storageManager: storageManager,
		syncManager:    syncManager,
	}
}

func (m mainMenuModel) Init() tea.Cmd {
	return m.checkOnlineStatus()
}

func (m mainMenuModel) checkOnlineStatus() tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return onlineStatusMsg{online: false}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		return onlineStatusMsg{online: m.client.IsServerAvailable(ctx)}
	}
}

// tickOnlineCheck periodically re-checks online status.
func tickOnlineCheck() tea.Cmd {
	return tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
		return recheckOnlineMsg{}
	})
}

type recheckOnlineMsg struct{}

func (m mainMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 4)
		return m, nil

	case onlineStatusMsg:
		m.isOnline = msg.online
		m.lastCheck = time.Now()
		return m, tickOnlineCheck()

	case recheckOnlineMsg:
		return m, m.checkOnlineStatus()

	case subProgramReturnedMsg:
		return m, m.checkOnlineStatus()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			selectedItem := m.list.SelectedItem()
			if selectedItem == nil {
				return m, nil
			}

			item := selectedItem.(menuItem)
			switch item.id {
			case "logout":
				// Очищаем токены и выходим
				username, err := m.tokenStore.GetCurrentUsername()
				if err == nil && username != "" {
					m.tokenStore.DeleteTokens(username)
				}
				m.quitting = true
				// Отправляем сообщение о logout для очистки ресурсов
				return m, tea.Sequence(
					func() tea.Msg { return logoutMsg{} },
					tea.Quit,
				)

			case "credentials":
				// Открываем экран credentials
				if m.offlineService == nil {
					return m, nil
				}
				credView := NewCredentialsViewModel(m.offlineService)
				return m, func() tea.Msg {
					tea.NewProgram(credView, tea.WithAltScreen()).Run()
					return subProgramReturnedMsg{}
				}

			case "cards":
				// Открываем экран cards
				if m.offlineService == nil {
					return m, nil
				}
				cardView := NewCardsViewModel(m.offlineService)
				return m, func() tea.Msg {
					tea.NewProgram(cardView, tea.WithAltScreen()).Run()
					return subProgramReturnedMsg{}
				}

			case "texts":
				// Открываем экран texts
				if m.offlineService == nil {
					return m, nil
				}
				textView := NewTextsViewModel(m.offlineService)
				return m, func() tea.Msg {
					tea.NewProgram(textView, tea.WithAltScreen()).Run()
					return subProgramReturnedMsg{}
				}

			case "sync":
				// Открываем экран синхронизации
				if m.storageManager == nil {
					return m, nil
				}
				syncView := NewSyncViewModel(m.storageManager, m.syncManager)
				return m, func() tea.Msg {
					tea.NewProgram(syncView, tea.WithAltScreen()).Run()
					return subProgramReturnedMsg{}
				}

			case "binaries":
				// Открываем экран binaries
				if m.storageManager == nil {
					return m, nil
				}
				binaryView := NewBinariesViewModel(m.storageManager)
				return m, func() tea.Msg {
					tea.NewProgram(binaryView, tea.WithAltScreen()).Run()
					return subProgramReturnedMsg{}
				}
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m mainMenuModel) View() string {
	if m.quitting {
		return ""
	}

	// Online/offline status indicator
	var statusLine string
	if m.isOnline {
		statusLine = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true).
			Render("  [ONLINE] Connected to server")
	} else {
		statusLine = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true).
			Render("  [OFFLINE] Working locally, changes will sync later")
	}

	help := menuHelpStyle.Render("\n  ↑/↓: navigate | Enter: select | q/Ctrl+C: exit")

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		m.list.View(),
		statusLine,
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
