package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/client/storage"
	"github.com/BigSm0uk/GophKeeper/internal/client/sync"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SyncViewModel manages synchronization view.
type SyncViewModel struct {
	storageManager *storage.StorageManager
	syncManager    *sync.Manager

	pendingCreds    []*storage.LocalCredential
	pendingBinaries []*storage.LocalBinary

	syncing     bool
	progress    progress.Model
	currentItem int
	totalItems  int

	err      error
	message  string
	quitting bool
}

// NewSyncViewModel creates a new sync view model.
func NewSyncViewModel(storageManager *storage.StorageManager, syncManager *sync.Manager) SyncViewModel {
	prog := progress.New(progress.WithDefaultGradient())

	return SyncViewModel{
		storageManager: storageManager,
		syncManager:    syncManager,
		progress:       prog,
	}
}

func (m SyncViewModel) Init() tea.Cmd {
	return m.loadPendingItems()
}

func (m SyncViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		// Во время синхронизации игнорируем все клавиши кроме ctrl+c
		if m.syncing {
			if key == "ctrl+c" {
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}

		// Обработка клавиш когда не синхронизируемся
		if key == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}

		if key == "esc" || key == "q" {
			// Возврат в главное меню
			m.quitting = true
			return m, tea.Quit
		}

		if key == "s" || key == "enter" {
			// Запустить синхронизацию
			if m.totalItems > 0 {
				return m, m.startSync()
			}
		}

	case pendingItemsLoadedMsg:
		m.pendingCreds = msg.creds
		m.pendingBinaries = msg.binaries
		m.err = msg.err
		m.totalItems = len(m.pendingCreds) + len(m.pendingBinaries)
		return m, nil

	case syncStartedMsg:
		m.syncing = true
		m.currentItem = 0
		return m, nil

	case syncProgressMsg:
		m.currentItem = msg.current
		if m.totalItems > 0 {
			cmd := m.progress.SetPercent(float64(m.currentItem) / float64(m.totalItems))
			return m, cmd
		}
		return m, nil

	case syncCompletedMsg:
		m.syncing = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.message = fmt.Sprintf("Sync completed! %d items synchronized.", msg.synced)
		}
		return m, m.loadPendingItems()

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd
	}

	return m, nil
}

func (m SyncViewModel) View() string {
	if m.quitting {
		return ""
	}

	if m.syncing {
		return m.viewSyncing()
	}

	return m.viewPending()
}

func (m SyncViewModel) viewPending() string {
	title := titleStyle.Render("🔄 Synchronization")

	var content string
	if m.err != nil {
		content = errorStyle.Render(fmt.Sprintf("❌ Error: %s", m.err.Error()))
	} else if m.totalItems == 0 {
		content = successStyle.Render("✅ All data is synchronized!")
	} else {
		stats := []string{
			fmt.Sprintf("Pending items: %d", m.totalItems),
			fmt.Sprintf("  • Credentials: %d", len(m.pendingCreds)),
			fmt.Sprintf("  • Binaries:    %d", len(m.pendingBinaries)),
		}

		statsText := lipgloss.JoinVertical(lipgloss.Left, stats...)

		info := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true).
			Render("\nThese items will be uploaded to the server.")

		content = lipgloss.JoinVertical(lipgloss.Left,
			statsText,
			info,
		)
	}

	help := menuHelpStyle.Render("\ns/Enter: start sync | Esc/q: back to menu")

	if m.message != "" {
		content = lipgloss.JoinVertical(lipgloss.Left,
			successStyle.Render("✅ "+m.message),
			"",
			content,
		)
	}

	view := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		"",
		content,
		help,
	)

	return lipgloss.Place(
		80,
		24,
		lipgloss.Center,
		lipgloss.Center,
		view,
	)
}

func (m SyncViewModel) viewSyncing() string {
	title := titleStyle.Render("🔄 Synchronizing...")

	statusText := fmt.Sprintf("Syncing item %d of %d", m.currentItem, m.totalItems)
	progressView := m.progress.View()

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		statusText,
		"",
		progressView,
		"",
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("Please wait... (Ctrl+C to force quit)"),
	)

	view := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		"",
		content,
	)

	return lipgloss.Place(
		80,
		24,
		lipgloss.Center,
		lipgloss.Center,
		view,
	)
}

// ====================
// Commands
// ====================

type pendingItemsLoadedMsg struct {
	creds    []*storage.LocalCredential
	binaries []*storage.LocalBinary
	err      error
}

type syncStartedMsg struct{}

type syncProgressMsg struct {
	current int
}

type syncCompletedMsg struct {
	synced int
	err    error
}

func (m SyncViewModel) loadPendingItems() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		creds, err := m.storageManager.Encrypted.GetPendingCredentials()
		if err != nil {
			return pendingItemsLoadedMsg{err: err}
		}

		binaries, err := m.storageManager.Encrypted.GetPendingBinaries()
		if err != nil {
			return pendingItemsLoadedMsg{err: err}
		}

		_ = ctx
		return pendingItemsLoadedMsg{
			creds:    creds,
			binaries: binaries,
		}
	}
}

func (m SyncViewModel) startSync() tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			return syncStartedMsg{}
		},
		func() tea.Msg {
			// Запускаем принудительную синхронизацию
			if m.syncManager != nil {
				m.syncManager.ForceSync()
			}

			// Эмулируем прогресс (в реальности sync manager работает асинхронно)
			// TODO: Интегрировать реальный прогресс от sync manager
			time.Sleep(2 * time.Second)

			return syncCompletedMsg{
				synced: m.totalItems,
			}
		},
	)
}
