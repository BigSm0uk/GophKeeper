package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/client/service"
	"github.com/BigSm0uk/GophKeeper/internal/client/storage"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type credViewMode int

const (
	credViewList credViewMode = iota
	credViewAdd
	credViewDetail
	credViewEdit
	credViewConfirmDelete
)

type credentialItem struct {
	cred *storage.LocalCredential
}

func (i credentialItem) FilterValue() string { return i.cred.Name }
func (i credentialItem) Title() string {
	statusIcon := getSyncStatusIcon(i.cred.SyncStatus)
	return fmt.Sprintf("%s %s", statusIcon, i.cred.Name)
}

func (i credentialItem) Description() string {
	return fmt.Sprintf("Login: %s | Updated: %s", i.cred.Login, i.cred.UpdatedAt.Format("2006-01-02 15:04"))
}

// CredentialsViewModel manages credentials CRUD operations.
type CredentialsViewModel struct {
	mode           credViewMode
	list           list.Model
	offlineService *service.OfflineService
	credentials    []*storage.LocalCredential
	selectedCred   *storage.LocalCredential

	// Form inputs
	nameInput     textinput.Model
	loginInput    textinput.Model
	passwordInput textinput.Model
	urlInput      textinput.Model
	metadataInput textinput.Model
	focusedField  int

	width  int
	height int

	err      error
	message  string
	quitting bool
	loading  bool
}

// NewCredentialsViewModel creates a new credentials view model.
func NewCredentialsViewModel(offlineService *service.OfflineService) CredentialsViewModel {
	items := []list.Item{}
	l := list.New(items, list.NewDefaultDelegate(), 80, 20)
	l.Title = "🔑 Credentials"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = menuTitleStyle

	// Инициализация полей ввода
	nameInput := textinput.New()
	nameInput.Placeholder = "Name (e.g., GitHub Account)"
	nameInput.Width = 50

	loginInput := textinput.New()
	loginInput.Placeholder = "Login/Username"
	loginInput.Width = 50

	passwordInput := textinput.New()
	passwordInput.Placeholder = "Password"
	passwordInput.EchoMode = textinput.EchoPassword
	passwordInput.Width = 50

	urlInput := textinput.New()
	urlInput.Placeholder = "URL (optional)"
	urlInput.Width = 50

	metadataInput := textinput.New()
	metadataInput.Placeholder = "Metadata (optional, JSON)"
	metadataInput.Width = 50

	return CredentialsViewModel{
		mode:           credViewList,
		list:           l,
		offlineService: offlineService,
		nameInput:      nameInput,
		loginInput:     loginInput,
		passwordInput:  passwordInput,
		urlInput:       urlInput,
		metadataInput:  metadataInput,
	}
}

func (m CredentialsViewModel) Init() tea.Cmd {
	return m.loadCredentials()
}

func (m CredentialsViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

	case tea.KeyMsg:
		key := msg.String()

		// Глобальные клавиши
		if key == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}

		// Обработка в зависимости от режима
		switch m.mode {
		case credViewList:
			if key == "esc" || key == "q" || key == "a" || key == "e" || key == "d" || key == "enter" {
				return m.handleListKeys(key)
			}
			var listCmd tea.Cmd
			m.list, listCmd = m.list.Update(msg)
			return m, listCmd
		case credViewAdd, credViewEdit:
			return m.handleFormKeys(msg)
		case credViewDetail:
			return m.handleDetailKeys(key)
		case credViewConfirmDelete:
			return m.handleConfirmKeys(key)
		}

	case credentialsLoadedMsg:
		m.loading = false
		m.credentials = msg.creds
		m.err = msg.err

		if m.err == nil {
			items := make([]list.Item, len(m.credentials))
			for i, cred := range m.credentials {
				items[i] = credentialItem{cred: cred}
			}
			m.list.SetItems(items)
		}
		return m, nil

	case credentialSavedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.message = "Credential saved successfully!"
		m.mode = credViewList
		return m, m.loadCredentials()

	case credentialDeletedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.message = "Credential deleted successfully!"
		m.mode = credViewList
		return m, m.loadCredentials()
	}

	return m, nil
}

func (m CredentialsViewModel) View() string {
	if m.quitting {
		return ""
	}

	switch m.mode {
	case credViewList:
		return m.viewList()
	case credViewAdd:
		return m.viewForm("Add Credential")
	case credViewEdit:
		return m.viewForm("Edit Credential")
	case credViewDetail:
		return m.viewDetail()
	case credViewConfirmDelete:
		return m.viewConfirmDelete()
	}

	return "Unknown view"
}

// ====================
// List View
// ====================

func (m CredentialsViewModel) viewList() string {
	if m.loading {
		return "Loading credentials..."
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

func (m *CredentialsViewModel) handleListKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		// Возврат в главное меню
		m.quitting = true
		return m, tea.Quit

	case "a":
		// Добавить новый credential
		m.mode = credViewAdd
		m.clearForm()
		m.focusedField = 0
		m.err = nil
		m.message = ""
		return m, m.nameInput.Focus()

	case "e":
		// Редактировать выбранный
		if len(m.credentials) == 0 {
			return m, nil
		}
		selectedItem := m.list.SelectedItem()
		if selectedItem == nil {
			return m, nil
		}
		item := selectedItem.(credentialItem)
		m.selectedCred = item.cred
		m.mode = credViewEdit
		m.loadFormFromCredential(m.selectedCred)
		m.focusedField = 0
		m.err = nil
		m.message = ""
		return m, m.nameInput.Focus()

	case "d":
		// Удалить выбранный
		if len(m.credentials) == 0 {
			return m, nil
		}
		selectedItem := m.list.SelectedItem()
		if selectedItem == nil {
			return m, nil
		}
		item := selectedItem.(credentialItem)
		m.selectedCred = item.cred
		m.mode = credViewConfirmDelete
		return m, nil

	case "enter":
		// Просмотр деталей
		if len(m.credentials) == 0 {
			return m, nil
		}
		selectedItem := m.list.SelectedItem()
		if selectedItem == nil {
			return m, nil
		}
		item := selectedItem.(credentialItem)
		m.selectedCred = item.cred
		m.mode = credViewDetail
		return m, nil
	}

	return m, nil
}

// ====================
// Form View (Add/Edit)
// ====================

func (m CredentialsViewModel) viewForm(title string) string {
	activeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true)
	inactiveStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	fields := []string{
		m.renderField("Name:", m.nameInput, 0, activeStyle, inactiveStyle),
		m.renderField("Login:", m.loginInput, 1, activeStyle, inactiveStyle),
		m.renderField("Password:", m.passwordInput, 2, activeStyle, inactiveStyle),
		m.renderField("URL:", m.urlInput, 3, activeStyle, inactiveStyle),
		m.renderField("Metadata:", m.metadataInput, 4, activeStyle, inactiveStyle),
	}

	form := lipgloss.JoinVertical(lipgloss.Left, fields...)

	status := ""
	if m.err != nil {
		status = "\n" + errorStyle.Render("❌ "+m.err.Error())
	}

	help := menuHelpStyle.Render("\nEnter: save | Tab: next field | Esc: cancel")

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		titleStyle.Render(title),
		"",
		form,
		status,
		help,
	)

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

func (m CredentialsViewModel) renderField(label string, input textinput.Model, fieldIndex int, activeStyle, inactiveStyle lipgloss.Style) string {
	labelText := label
	if m.focusedField == fieldIndex {
		labelText = "→ " + activeStyle.Render(label)
	} else {
		labelText = "  " + inactiveStyle.Render(label)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		labelText,
		"  "+input.View(),
		"",
	)
}

func (m *CredentialsViewModel) handleFormKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "esc":
		m.mode = credViewList
		m.clearForm()
		m.err = nil
		return m, nil

	case "tab", "shift+tab":
		m.err = nil
		if key == "tab" {
			m.focusedField = (m.focusedField + 1) % 5
		} else {
			m.focusedField = (m.focusedField - 1 + 5) % 5
		}
		return m, m.focusField()

	case "enter":
		return m, m.saveCredential()

	case "up", "down":
		return m.updateActiveInput(msg)
	}

	return m.updateActiveInput(msg)
}

func (m *CredentialsViewModel) focusField() tea.Cmd {
	inputs := []*textinput.Model{
		&m.nameInput,
		&m.loginInput,
		&m.passwordInput,
		&m.urlInput,
		&m.metadataInput,
	}

	for i, input := range inputs {
		if i == m.focusedField {
			return input.Focus()
		}
		input.Blur()
	}

	return nil
}

func (m *CredentialsViewModel) updateActiveInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.focusedField {
	case 0:
		m.nameInput, cmd = m.nameInput.Update(msg)
	case 1:
		m.loginInput, cmd = m.loginInput.Update(msg)
	case 2:
		m.passwordInput, cmd = m.passwordInput.Update(msg)
	case 3:
		m.urlInput, cmd = m.urlInput.Update(msg)
	case 4:
		m.metadataInput, cmd = m.metadataInput.Update(msg)
	}

	return m, cmd
}

func (m *CredentialsViewModel) clearForm() {
	m.nameInput.SetValue("")
	m.loginInput.SetValue("")
	m.passwordInput.SetValue("")
	m.urlInput.SetValue("")
	m.metadataInput.SetValue("")
	m.nameInput.Blur()
	m.loginInput.Blur()
	m.passwordInput.Blur()
	m.urlInput.Blur()
	m.metadataInput.Blur()
	m.focusedField = 0
}

func (m *CredentialsViewModel) loadFormFromCredential(cred *storage.LocalCredential) {
	m.nameInput.SetValue(cred.Name)
	m.loginInput.SetValue(cred.Login)
	m.passwordInput.SetValue(cred.Password)

	if cred.URL != nil {
		m.urlInput.SetValue(*cred.URL)
	}
	if cred.Metadata != nil {
		m.metadataInput.SetValue(*cred.Metadata)
	}
}

// ====================
// Detail View
// ====================

func (m CredentialsViewModel) viewDetail() string {
	if m.selectedCred == nil {
		return "No credential selected"
	}

	cred := m.selectedCred

	details := []string{
		fmt.Sprintf("Name:       %s", cred.Name),
		fmt.Sprintf("Login:      %s", cred.Login),
		fmt.Sprintf("Password:   %s", cred.Password),
	}

	if cred.URL != nil && *cred.URL != "" {
		details = append(details, fmt.Sprintf("URL:        %s", *cred.URL))
	}

	if cred.Metadata != nil && *cred.Metadata != "" {
		details = append(details, fmt.Sprintf("Metadata:   %s", *cred.Metadata))
	}

	details = append(details,
		"",
		fmt.Sprintf("Status:     %s", GetSyncStatusStyled(string(cred.SyncStatus))),
		fmt.Sprintf("Created:    %s", cred.CreatedAt.Format("2006-01-02 15:04:05")),
		fmt.Sprintf("Updated:    %s", cred.UpdatedAt.Format("2006-01-02 15:04:05")),
	)

	if cred.SyncedAt != nil {
		details = append(details, fmt.Sprintf("Synced:     %s", cred.SyncedAt.Format("2006-01-02 15:04:05")))
	}

	detailsText := lipgloss.JoinVertical(lipgloss.Left, details...)

	help := menuHelpStyle.Render("\ne: edit | d: delete | Esc: back")

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		titleStyle.Render("🔑 Credential Details"),
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
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, content)
}

func (m *CredentialsViewModel) handleDetailKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		m.mode = credViewList
		m.selectedCred = nil
		return m, nil

	case "e":
		// Перейти к редактированию
		m.mode = credViewEdit
		m.loadFormFromCredential(m.selectedCred)
		m.focusedField = 0
		return m, m.nameInput.Focus()

	case "d":
		// Перейти к подтверждению удаления
		m.mode = credViewConfirmDelete
		return m, nil
	}

	return m, nil
}

// ====================
// Delete Confirmation
// ====================

func (m CredentialsViewModel) viewConfirmDelete() string {
	if m.selectedCred == nil {
		return "No credential selected"
	}

	warning := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true).
		Render(fmt.Sprintf("⚠️  Delete '%s'?", m.selectedCred.Name))

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

func (m *CredentialsViewModel) handleConfirmKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "y":
		// Подтверждение удаления
		return m, m.deleteCredential()

	case "n", "esc":
		// Отмена
		m.mode = credViewList
		m.selectedCred = nil
		return m, nil
	}

	return m, nil
}

// ====================
// Commands
// ====================

type credentialsLoadedMsg struct {
	creds []*storage.LocalCredential
	err   error
}

type credentialSavedMsg struct {
	err error
}

type credentialDeletedMsg struct {
	err error
}

func (m CredentialsViewModel) loadCredentials() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		creds, err := m.offlineService.ListCredentials(ctx)
		return credentialsLoadedMsg{creds: creds, err: err}
	}
}

func (m *CredentialsViewModel) saveCredential() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		name := m.nameInput.Value()
		login := m.loginInput.Value()
		password := m.passwordInput.Value()

		if name == "" || login == "" || password == "" {
			return credentialSavedMsg{err: fmt.Errorf("name, login, and password are required")}
		}

		var url, metadata *string
		if m.urlInput.Value() != "" {
			val := m.urlInput.Value()
			url = &val
		}
		if m.metadataInput.Value() != "" {
			val := m.metadataInput.Value()
			metadata = &val
		}

		var err error
		if m.mode == credViewAdd {
			_, err = m.offlineService.CreateCredential(ctx, name, login, password, url, metadata)
		} else if m.mode == credViewEdit && m.selectedCred != nil {
			_, err = m.offlineService.UpdateCredential(ctx, m.selectedCred.ID, name, login, password, url, metadata)
		}

		return credentialSavedMsg{err: err}
	}
}

func (m *CredentialsViewModel) deleteCredential() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if m.selectedCred == nil {
			return credentialDeletedMsg{err: fmt.Errorf("no credential selected")}
		}

		err := m.offlineService.DeleteCredential(ctx, m.selectedCred.ID)
		return credentialDeletedMsg{err: err}
	}
}

// ====================
// Helpers
// ====================

func getSyncStatusIcon(status storage.SyncStatus) string {
	switch status {
	case storage.StatusSynced:
		return "✓"
	case storage.StatusPending:
		return "⏳"
	case storage.StatusUploading:
		return "📤"
	case storage.StatusUpdated:
		return "✎"
	case storage.StatusDeleted:
		return "🗑"
	default:
		return "?"
	}
}

// Placeholder for unused variable fix
var _ = getSyncStatusIcon
