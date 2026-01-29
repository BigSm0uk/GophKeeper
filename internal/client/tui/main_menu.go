package tui

import (
	"github.com/BigSm0uk/GophKeeper/internal/client/api"
	"github.com/BigSm0uk/GophKeeper/internal/client/storage"
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
	list       list.Model
	client     *api.Client
	tokenStore *storage.TokenStore
	quitting   bool
}

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

func NewMainMenuModel(client *api.Client, tokenStore *storage.TokenStore) mainMenuModel {
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
		list:       l,
		client:     client,
		tokenStore: tokenStore,
	}
}

func (m mainMenuModel) Init() tea.Cmd {
	return nil
}

func (m mainMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 4)
		return m, nil

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
				return m, tea.Quit

			case "credentials", "texts", "cards", "binaries", "sync":
				// TODO: Переход на соответствующие экраны
				// Пока просто показываем сообщение
				return m, nil
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

	help := menuHelpStyle.Render("\n  ↑/↓: navigate | Enter: select | q/Ctrl+C: exit")

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		m.list.View(),
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
