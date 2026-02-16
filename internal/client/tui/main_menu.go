package tui

import (
	"context"
	"io"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/client/api"
	"github.com/BigSm0uk/GophKeeper/internal/client/service"
	"github.com/BigSm0uk/GophKeeper/internal/client/storage"
	"github.com/BigSm0uk/GophKeeper/internal/client/sync"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// teaSubCmd wraps a tea.Model to run as a sub-program via tea.Exec.
// This ensures the parent program properly suspends while the child runs,
// preventing input competition and rendering conflicts.
type teaSubCmd struct {
	model  tea.Model
	opts   []tea.ProgramOption
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

func (c *teaSubCmd) SetStdin(r io.Reader)  { c.stdin = r }
func (c *teaSubCmd) SetStdout(w io.Writer) { c.stdout = w }
func (c *teaSubCmd) SetStderr(w io.Writer) { c.stderr = w }

// Run creates and runs a new tea.Program with the parent's I/O.
func (c *teaSubCmd) Run() error {
	opts := make([]tea.ProgramOption, len(c.opts))
	copy(opts, c.opts)
	opts = append(opts, tea.WithInput(c.stdin), tea.WithOutput(c.stdout))
	_, err := tea.NewProgram(c.model, opts...).Run()
	return err
}

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
	l.SetShowHelp(false)
	l.DisableQuitKeybindings()
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

	case tea.MouseMsg:
		// Block all mouse events to prevent unintended navigation
		return m, nil

	case onlineStatusMsg:
		m.isOnline = msg.online
		m.lastCheck = time.Now()
		return m, tickOnlineCheck()

	case recheckOnlineMsg:
		return m, m.checkOnlineStatus()

	case subProgramReturnedMsg:
		// Re-query window size to force a full re-render after returning from sub-program
		return m, tea.Batch(m.checkOnlineStatus(), tea.WindowSize())

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
				if m.offlineService == nil {
					return m, nil
				}
				return m, tea.Exec(&teaSubCmd{
					model: NewCredentialsViewModel(m.offlineService),
					opts:  []tea.ProgramOption{tea.WithAltScreen()},
				}, func(err error) tea.Msg {
					return subProgramReturnedMsg{}
				})

			case "cards":
				if m.offlineService == nil {
					return m, nil
				}
				return m, tea.Exec(&teaSubCmd{
					model: NewCardsViewModel(m.offlineService),
					opts:  []tea.ProgramOption{tea.WithAltScreen()},
				}, func(err error) tea.Msg {
					return subProgramReturnedMsg{}
				})

			case "texts":
				if m.offlineService == nil {
					return m, nil
				}
				return m, tea.Exec(&teaSubCmd{
					model: NewTextsViewModel(m.offlineService),
					opts:  []tea.ProgramOption{tea.WithAltScreen()},
				}, func(err error) tea.Msg {
					return subProgramReturnedMsg{}
				})

			case "sync":
				if m.storageManager == nil {
					return m, nil
				}
				return m, tea.Exec(&teaSubCmd{
					model: NewSyncViewModel(m.storageManager, m.syncManager),
					opts:  []tea.ProgramOption{tea.WithAltScreen()},
				}, func(err error) tea.Msg {
					return subProgramReturnedMsg{}
				})

			case "binaries":
				if m.storageManager == nil {
					return m, nil
				}
				return m, tea.Exec(&teaSubCmd{
					model: NewBinariesViewModel(m.storageManager),
					opts:  []tea.ProgramOption{tea.WithAltScreen()},
				}, func(err error) tea.Msg {
					return subProgramReturnedMsg{}
				})
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
