package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/client/service"
	"github.com/BigSm0uk/GophKeeper/internal/client/storage"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type textViewMode int

const (
	textViewList textViewMode = iota
	textViewAdd
	textViewDetail
	textViewEdit
	textViewConfirmDelete
)

type textItem struct {
	text *storage.LocalText
}

func (i textItem) FilterValue() string { return i.text.Name }
func (i textItem) Title() string {
	statusIcon := getSyncStatusIcon(i.text.SyncStatus)
	return fmt.Sprintf("%s %s", statusIcon, i.text.Name)
}

func (i textItem) Description() string {
	preview := strings.ReplaceAll(i.text.Content, "\n", " ")
	if len(preview) > 60 {
		preview = preview[:60] + "..."
	}
	return fmt.Sprintf("%s | Updated: %s", preview, i.text.UpdatedAt.Format("2006-01-02 15:04"))
}

// TextsViewModel manages texts CRUD operations.
type TextsViewModel struct {
	mode           textViewMode
	list           list.Model
	offlineService *service.OfflineService
	texts          []*storage.LocalText
	selectedText   *storage.LocalText

	// Form inputs
	nameInput     textinput.Model
	contentArea   textarea.Model
	metadataInput textinput.Model
	focusedField  int

	// Window size for layout (form fits on screen)
	width  int
	height int

	err      error
	message  string
	quitting bool
	loading  bool
}

// NewTextsViewModel creates a new texts view model.
func NewTextsViewModel(offlineService *service.OfflineService) TextsViewModel {
	items := []list.Item{}
	l := list.New(items, list.NewDefaultDelegate(), 80, 20)
	l.Title = "📝 Text Notes"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.DisableQuitKeybindings()
	l.Styles.Title = menuTitleStyle

	nameInput := textinput.New()
	nameInput.Placeholder = "Note name (e.g., Meeting Notes)"
	nameInput.Width = 60

	contentArea := textarea.New()
	contentArea.Placeholder = "Enter your text here..."
	contentArea.SetWidth(60)
	contentArea.SetHeight(10)
	contentArea.CharLimit = 10000

	metadataInput := textinput.New()
	metadataInput.Placeholder = "Metadata (optional, JSON)"
	metadataInput.Width = 60

	return TextsViewModel{
		mode:           textViewList,
		list:           l,
		offlineService: offlineService,
		nameInput:      nameInput,
		contentArea:    contentArea,
		metadataInput:  metadataInput,
	}
}

func (m TextsViewModel) Init() tea.Cmd {
	return m.loadTexts()
}

func (m TextsViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Если уже выходим, игнорируем все сообщения кроме KeyMsg
	if m.quitting {
		if _, ok := msg.(tea.KeyMsg); !ok {
			return m, nil
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 4)
		return m, nil

	case tea.MouseMsg:
		// Block all mouse events to prevent unintended navigation
		return m, nil

	case tea.KeyMsg:
		key := msg.String()

		if key == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}

		switch m.mode {
		case textViewList:
			// Action keys handled by handleListKeys; up/down and others go to list
			if key == "esc" || key == "q" || key == "a" || key == "e" || key == "d" || key == "enter" {
				return m.handleListKeys(key)
			}
			var listCmd tea.Cmd
			m.list, listCmd = m.list.Update(msg)
			return m, listCmd
		case textViewAdd, textViewEdit:
			return m.handleFormKeys(msg)
		case textViewDetail:
			return m.handleDetailKeys(key)
		case textViewConfirmDelete:
			return m.handleConfirmKeys(key)
		}

	case textsLoadedMsg:
		m.loading = false
		m.texts = msg.texts
		m.err = msg.err

		if m.err == nil {
			items := make([]list.Item, len(m.texts))
			for i, text := range m.texts {
				items[i] = textItem{text: text}
			}
			m.list.SetItems(items)
		}
		return m, nil

	case textSavedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.message = "Text note saved successfully!"
		m.mode = textViewList
		return m, m.loadTexts()

	case textDeletedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.message = "Text note deleted successfully!"
		m.mode = textViewList
		return m, m.loadTexts()
	}

	return m, nil
}

func (m TextsViewModel) View() string {
	if m.quitting {
		return ""
	}

	switch m.mode {
	case textViewList:
		return m.viewList()
	case textViewAdd:
		return m.viewForm("Add Text Note")
	case textViewEdit:
		return m.viewForm("Edit Text Note")
	case textViewDetail:
		return m.viewDetail()
	case textViewConfirmDelete:
		return m.viewConfirmDelete()
	}

	return "Unknown view"
}

func (m TextsViewModel) viewList() string {
	if m.loading {
		return "Loading text notes..."
	}

	status := ""
	if m.err != nil {
		status = errorStyle.Render("\n❌ Error: " + m.err.Error())
	} else if m.message != "" {
		status = successStyle.Render("\n✅ " + m.message)
	}

	help := menuHelpStyle.Render("\n↑/↓: navigate | Enter: view | a: add | e: edit | d: delete | Esc: back")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.list.View(),
		status,
		help,
	)
}

func (m *TextsViewModel) handleListKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		// Возврат в главное меню
		m.quitting = true
		return m, tea.Quit

	case "a":
		m.mode = textViewAdd
		m.clearForm()
		m.focusedField = 0
		m.err = nil
		m.message = ""
		return m, m.nameInput.Focus()

	case "e":
		if len(m.texts) == 0 {
			return m, nil
		}
		selectedItem := m.list.SelectedItem()
		if selectedItem == nil {
			return m, nil
		}
		item := selectedItem.(textItem)
		m.selectedText = item.text
		m.mode = textViewEdit
		m.loadFormFromText(m.selectedText)
		m.focusedField = 0
		m.err = nil
		m.message = ""
		return m, m.nameInput.Focus()

	case "d":
		if len(m.texts) == 0 {
			return m, nil
		}
		selectedItem := m.list.SelectedItem()
		if selectedItem == nil {
			return m, nil
		}
		item := selectedItem.(textItem)
		m.selectedText = item.text
		m.mode = textViewConfirmDelete
		return m, nil

	case "enter":
		if len(m.texts) == 0 {
			return m, nil
		}
		selectedItem := m.list.SelectedItem()
		if selectedItem == nil {
			return m, nil
		}
		item := selectedItem.(textItem)
		m.selectedText = item.text
		m.mode = textViewDetail
		return m, nil
	}

	return m, nil
}

func (m TextsViewModel) viewForm(title string) string {
	activeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true)
	inactiveStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	var nameLabel, contentLabel, metadataLabel string

	if m.focusedField == 0 {
		nameLabel = "→ " + activeStyle.Render("Name:")
	} else {
		nameLabel = "  " + inactiveStyle.Render("Name:")
	}

	if m.focusedField == 1 {
		contentLabel = "→ " + activeStyle.Render("Content:")
	} else {
		contentLabel = "  " + inactiveStyle.Render("Content:")
	}

	if m.focusedField == 2 {
		metadataLabel = "→ " + activeStyle.Render("Metadata:")
	} else {
		metadataLabel = "  " + inactiveStyle.Render("Metadata:")
	}

	form := lipgloss.JoinVertical(lipgloss.Left,
		nameLabel,
		"  "+m.nameInput.View(),
		"",
		contentLabel,
		"  "+m.contentArea.View(),
		"",
		metadataLabel,
		"  "+m.metadataInput.View(),
	)

	status := ""
	if m.err != nil {
		status = "\n" + errorStyle.Render("❌ "+m.err.Error())
	}

	help := menuHelpStyle.Render("\nCtrl+S: save | Tab: next field | Esc: cancel")

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		titleStyle.Render(title),
		"",
		form,
		status,
		help,
	)

	// Use actual window size so all fields fit on screen (no fixed 40-line box)
	w, h := m.width, m.height
	if w <= 0 {
		w = 100
	}
	if h <= 0 {
		h = 24
	}
	return lipgloss.Place(
		w,
		h,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

func (m *TextsViewModel) handleFormKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "esc":
		m.mode = textViewList
		m.clearForm()
		m.err = nil
		return m, nil

	case "tab":
		m.err = nil
		m.focusedField = (m.focusedField + 1) % 3
		return m, m.focusField()

	case "shift+tab":
		m.err = nil
		m.focusedField = (m.focusedField - 1 + 3) % 3
		return m, m.focusField()

	case "ctrl+s":
		return m, m.saveText()

	case "up", "down":
		// Arrows go only to the active field (e.g. textarea line navigation), never switch to list
		return m.updateActiveInput(msg)
	}

	return m.updateActiveInput(msg)
}

func (m *TextsViewModel) focusField() tea.Cmd {
	m.nameInput.Blur()
	m.contentArea.Blur()
	m.metadataInput.Blur()

	switch m.focusedField {
	case 0:
		return m.nameInput.Focus()
	case 1:
		m.contentArea.Focus()
		return nil
	case 2:
		return m.metadataInput.Focus()
	}

	return nil
}

func (m *TextsViewModel) updateActiveInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.focusedField {
	case 0:
		m.nameInput, cmd = m.nameInput.Update(msg)
	case 1:
		m.contentArea, cmd = m.contentArea.Update(msg)
	case 2:
		m.metadataInput, cmd = m.metadataInput.Update(msg)
	}

	return m, cmd
}

func (m *TextsViewModel) clearForm() {
	m.nameInput.SetValue("")
	m.contentArea.SetValue("")
	m.metadataInput.SetValue("")
	m.nameInput.Blur()
	m.contentArea.Blur()
	m.metadataInput.Blur()
	m.focusedField = 0
}

func (m *TextsViewModel) loadFormFromText(text *storage.LocalText) {
	m.nameInput.SetValue(text.Name)
	m.contentArea.SetValue(text.Content)

	if text.Metadata != nil {
		m.metadataInput.SetValue(*text.Metadata)
	}
}

func (m TextsViewModel) viewDetail() string {
	if m.selectedText == nil {
		return "No text selected"
	}

	text := m.selectedText

	// Форматируем контент с переносами строк
	contentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Width(80).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1)

	details := []string{
		fmt.Sprintf("Name:       %s", text.Name),
		"",
		"Content:",
		contentStyle.Render(text.Content),
		"",
	}

	if text.Metadata != nil && *text.Metadata != "" {
		details = append(details, fmt.Sprintf("Metadata:   %s", *text.Metadata))
		details = append(details, "")
	}

	details = append(details,
		fmt.Sprintf("Status:     %s", GetSyncStatusStyled(string(text.SyncStatus))),
		fmt.Sprintf("Created:    %s", text.CreatedAt.Format("2006-01-02 15:04:05")),
		fmt.Sprintf("Updated:    %s", text.UpdatedAt.Format("2006-01-02 15:04:05")),
	)

	if text.SyncedAt != nil {
		details = append(details, fmt.Sprintf("Synced:     %s", text.SyncedAt.Format("2006-01-02 15:04:05")))
	}

	detailsText := lipgloss.JoinVertical(lipgloss.Left, details...)
	help := menuHelpStyle.Render("\ne: edit | d: delete | Esc: back")

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		titleStyle.Render("📝 Text Note Details"),
		"",
		detailsText,
		help,
	)

	w, h := m.width, m.height
	if w <= 0 {
		w = 100
	}
	if h <= 0 {
		h = 24
	}
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Top, content)
}

func (m *TextsViewModel) handleDetailKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		m.mode = textViewList
		m.selectedText = nil
		return m, nil

	case "e":
		m.mode = textViewEdit
		m.loadFormFromText(m.selectedText)
		m.focusedField = 0
		return m, m.nameInput.Focus()

	case "d":
		m.mode = textViewConfirmDelete
		return m, nil
	}

	return m, nil
}

func (m TextsViewModel) viewConfirmDelete() string {
	if m.selectedText == nil {
		return "No text selected"
	}

	warning := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true).
		Render(fmt.Sprintf("⚠️  Delete '%s'?", m.selectedText.Name))

	info := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render("This action cannot be undone.")

	help := menuHelpStyle.Render("\ny: confirm | n/Esc: cancel")

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		titleStyle.Render("🗑️  Confirm Deletion"),
		"",
		warning,
		"",
		info,
		help,
	)

	w, h := m.width, m.height
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, content)
}

func (m *TextsViewModel) handleConfirmKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "y":
		return m, m.deleteText()
	case "n", "esc":
		m.mode = textViewList
		m.selectedText = nil
		return m, nil
	}
	return m, nil
}

type textsLoadedMsg struct {
	texts []*storage.LocalText
	err   error
}

type textSavedMsg struct {
	err error
}

type textDeletedMsg struct {
	err error
}

func (m TextsViewModel) loadTexts() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		texts, err := m.offlineService.ListTexts(ctx)
		return textsLoadedMsg{texts: texts, err: err}
	}
}

func (m *TextsViewModel) saveText() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		name := m.nameInput.Value()
		content := m.contentArea.Value()

		if name == "" || content == "" {
			return textSavedMsg{err: fmt.Errorf("name and content are required")}
		}

		var metadata *string
		if m.metadataInput.Value() != "" {
			val := m.metadataInput.Value()
			metadata = &val
		}

		var err error
		if m.mode == textViewAdd {
			_, err = m.offlineService.CreateText(ctx, name, content, metadata)
		} else if m.mode == textViewEdit && m.selectedText != nil {
			_, err = m.offlineService.UpdateText(ctx, m.selectedText.ID, name, content, metadata)
		}

		return textSavedMsg{err: err}
	}
}

func (m *TextsViewModel) deleteText() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if m.selectedText == nil {
			return textDeletedMsg{err: fmt.Errorf("no text selected")}
		}

		err := m.offlineService.DeleteText(ctx, m.selectedText.ID)
		return textDeletedMsg{err: err}
	}
}
