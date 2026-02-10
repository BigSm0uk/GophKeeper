package tui

import (
	"context"
	"fmt"
	"mime"
	"path/filepath"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/client/storage"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type binaryViewMode int

const (
	binaryViewList binaryViewMode = iota
	binaryViewAdd
	binaryViewDetail
	binaryViewConfirmDelete
)

type binaryItem struct {
	binary *storage.LocalBinary
}

func (i binaryItem) FilterValue() string { return i.binary.Name }
func (i binaryItem) Title() string {
	statusIcon := getSyncStatusIcon(i.binary.SyncStatus)
	size := formatFileSize(i.binary.Size)
	return fmt.Sprintf("%s %s (%s)", statusIcon, i.binary.Name, size)
}

func (i binaryItem) Description() string {
	return fmt.Sprintf("File: %s | Type: %s | Updated: %s",
		i.binary.Filename,
		i.binary.ContentType,
		i.binary.UpdatedAt.Format("2006-01-02 15:04"))
}

// BinariesViewModel manages binaries CRUD operations.
type BinariesViewModel struct {
	mode           binaryViewMode
	list           list.Model
	storageManager *storage.StorageManager
	binaries       []*storage.LocalBinary
	selectedBinary *storage.LocalBinary

	// Form inputs
	nameInput     textinput.Model
	filePathInput textinput.Model
	metadataInput textinput.Model
	focusedField  int

	width  int
	height int

	err      error
	message  string
	quitting bool
	loading  bool
}

// NewBinariesViewModel creates a new binaries view model.
func NewBinariesViewModel(storageManager *storage.StorageManager) BinariesViewModel {
	items := []list.Item{}
	l := list.New(items, list.NewDefaultDelegate(), 80, 20)
	l.Title = "📁 Binary Files"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = menuTitleStyle

	nameInput := textinput.New()
	nameInput.Placeholder = "File name (e.g., Passport Scan)"
	nameInput.Width = 60

	filePathInput := textinput.New()
	filePathInput.Placeholder = "File path (e.g., /path/to/file.pdf)"
	filePathInput.Width = 60

	metadataInput := textinput.New()
	metadataInput.Placeholder = "Metadata (optional, JSON)"
	metadataInput.Width = 60

	return BinariesViewModel{
		mode:           binaryViewList,
		list:           l,
		storageManager: storageManager,
		nameInput:      nameInput,
		filePathInput:  filePathInput,
		metadataInput:  metadataInput,
	}
}

func (m BinariesViewModel) Init() tea.Cmd {
	return m.loadBinaries()
}

func (m BinariesViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

		if key == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}

		switch m.mode {
		case binaryViewList:
			if key == "esc" || key == "q" || key == "u" || key == "d" || key == "enter" {
				return m.handleListKeys(key)
			}
			var listCmd tea.Cmd
			m.list, listCmd = m.list.Update(msg)
			return m, listCmd
		case binaryViewAdd:
			return m.handleFormKeys(msg)
		case binaryViewDetail:
			return m.handleDetailKeys(key)
		case binaryViewConfirmDelete:
			return m.handleConfirmKeys(key)
		}

	case binariesLoadedMsg:
		m.loading = false
		m.binaries = msg.binaries
		m.err = msg.err

		if m.err == nil {
			items := make([]list.Item, len(m.binaries))
			for i, binary := range m.binaries {
				items[i] = binaryItem{binary: binary}
			}
			m.list.SetItems(items)
		}
		return m, nil

	case binarySavedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.message = "File uploaded successfully!"
		m.mode = binaryViewList
		return m, m.loadBinaries()

	case binaryDeletedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.message = "File deleted successfully!"
		m.mode = binaryViewList
		return m, m.loadBinaries()
	}

	return m, nil
}

func (m BinariesViewModel) View() string {
	if m.quitting {
		return ""
	}

	switch m.mode {
	case binaryViewList:
		return m.viewList()
	case binaryViewAdd:
		return m.viewForm("Upload File")
	case binaryViewDetail:
		return m.viewDetail()
	case binaryViewConfirmDelete:
		return m.viewConfirmDelete()
	}

	return "Unknown view"
}

func (m BinariesViewModel) viewList() string {
	if m.loading {
		return "Loading files..."
	}

	status := ""
	if m.err != nil {
		status = errorStyle.Render("\n❌ Error: " + m.err.Error())
	} else if m.message != "" {
		status = successStyle.Render("\n✅ " + m.message)
	}

	help := menuHelpStyle.Render("\n↑/↓: navigate | Enter: view | u: upload | d: delete | Esc: back")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.list.View(),
		status,
		help,
	)
}

func (m *BinariesViewModel) handleListKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		// Возврат в главное меню
		m.quitting = true
		return m, tea.Quit

	case "u":
		m.mode = binaryViewAdd
		m.clearForm()
		m.focusedField = 0
		m.err = nil
		m.message = ""
		return m, m.nameInput.Focus()

	case "d":
		if len(m.binaries) == 0 {
			return m, nil
		}
		selectedItem := m.list.SelectedItem()
		if selectedItem == nil {
			return m, nil
		}
		item := selectedItem.(binaryItem)
		m.selectedBinary = item.binary
		m.mode = binaryViewConfirmDelete
		return m, nil

	case "enter":
		if len(m.binaries) == 0 {
			return m, nil
		}
		selectedItem := m.list.SelectedItem()
		if selectedItem == nil {
			return m, nil
		}
		item := selectedItem.(binaryItem)
		m.selectedBinary = item.binary
		m.mode = binaryViewDetail
		return m, nil
	}

	return m, nil
}

func (m BinariesViewModel) viewForm(title string) string {
	activeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true)
	inactiveStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	fields := []string{
		m.renderField("File Name:", m.nameInput, 0, activeStyle, inactiveStyle),
		m.renderField("File Path:", m.filePathInput, 1, activeStyle, inactiveStyle),
		m.renderField("Metadata:", m.metadataInput, 2, activeStyle, inactiveStyle),
	}

	form := lipgloss.JoinVertical(lipgloss.Left, fields...)

	status := ""
	if m.err != nil {
		status = "\n" + errorStyle.Render("❌ "+m.err.Error())
	}

	help := menuHelpStyle.Render("\nEnter: upload | Tab: next field | Esc: cancel")

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
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, content)
}

func (m BinariesViewModel) renderField(label string, input textinput.Model, fieldIndex int, activeStyle, inactiveStyle lipgloss.Style) string {
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

func (m *BinariesViewModel) handleFormKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "esc":
		m.mode = binaryViewList
		m.clearForm()
		m.err = nil
		return m, nil

	case "tab", "shift+tab":
		m.err = nil
		if key == "tab" {
			m.focusedField = (m.focusedField + 1) % 3
		} else {
			m.focusedField = (m.focusedField - 1 + 3) % 3
		}
		return m, m.focusField()

	case "enter":
		return m, m.uploadFile()

	case "up", "down":
		return m.updateActiveInput(msg)
	}

	return m.updateActiveInput(msg)
}

func (m *BinariesViewModel) focusField() tea.Cmd {
	inputs := []*textinput.Model{
		&m.nameInput,
		&m.filePathInput,
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

func (m *BinariesViewModel) updateActiveInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.focusedField {
	case 0:
		m.nameInput, cmd = m.nameInput.Update(msg)
	case 1:
		m.filePathInput, cmd = m.filePathInput.Update(msg)
	case 2:
		m.metadataInput, cmd = m.metadataInput.Update(msg)
	}

	return m, cmd
}

func (m *BinariesViewModel) clearForm() {
	m.nameInput.SetValue("")
	m.filePathInput.SetValue("")
	m.metadataInput.SetValue("")
	m.nameInput.Blur()
	m.filePathInput.Blur()
	m.metadataInput.Blur()
	m.focusedField = 0
}

func (m BinariesViewModel) viewDetail() string {
	if m.selectedBinary == nil {
		return "No file selected"
	}

	binary := m.selectedBinary

	details := []string{
		fmt.Sprintf("Name:         %s", binary.Name),
		fmt.Sprintf("Filename:     %s", binary.Filename),
		fmt.Sprintf("Size:         %s", formatFileSize(binary.Size)),
		fmt.Sprintf("Type:         %s", binary.ContentType),
		fmt.Sprintf("Checksum:     %s", binary.Checksum[:16]+"..."),
		fmt.Sprintf("Local Path:   %s", binary.FilePath),
	}

	if binary.Metadata != nil && *binary.Metadata != "" {
		details = append(details, fmt.Sprintf("Metadata:     %s", *binary.Metadata))
	}

	details = append(details,
		"",
		fmt.Sprintf("Status:       %s", GetSyncStatusStyled(string(binary.SyncStatus))),
		fmt.Sprintf("Created:      %s", binary.CreatedAt.Format("2006-01-02 15:04:05")),
		fmt.Sprintf("Updated:      %s", binary.UpdatedAt.Format("2006-01-02 15:04:05")),
	)

	if binary.SyncedAt != nil {
		details = append(details, fmt.Sprintf("Synced:       %s", binary.SyncedAt.Format("2006-01-02 15:04:05")))
	}

	detailsText := lipgloss.JoinVertical(lipgloss.Left, details...)
	help := menuHelpStyle.Render("\nd: delete | e: export | Esc: back")

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		titleStyle.Render("📁 File Details"),
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

func (m *BinariesViewModel) handleDetailKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		m.mode = binaryViewList
		m.selectedBinary = nil
		return m, nil

	case "d":
		m.mode = binaryViewConfirmDelete
		return m, nil

	case "e":
		// TODO: Export file - показать диалог выбора места сохранения
		m.message = "Export functionality coming soon..."
		return m, nil
	}

	return m, nil
}

func (m BinariesViewModel) viewConfirmDelete() string {
	if m.selectedBinary == nil {
		return "No file selected"
	}

	warning := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true).
		Render(fmt.Sprintf("⚠️  Delete file '%s'?", m.selectedBinary.Name))

	info := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render("This will delete both the encrypted file and metadata.")

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

func (m *BinariesViewModel) handleConfirmKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "y":
		return m, m.deleteBinary()
	case "n", "esc":
		m.mode = binaryViewList
		m.selectedBinary = nil
		return m, nil
	}
	return m, nil
}

// ====================
// Commands
// ====================

type binariesLoadedMsg struct {
	binaries []*storage.LocalBinary
	err      error
}

type binarySavedMsg struct {
	err error
}

type binaryDeletedMsg struct {
	err error
}

func (m BinariesViewModel) loadBinaries() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		binaries, err := m.storageManager.Encrypted.ListBinaries()
		_ = ctx
		return binariesLoadedMsg{binaries: binaries, err: err}
	}
}

func (m *BinariesViewModel) uploadFile() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		name := m.nameInput.Value()
		filePath := m.filePathInput.Value()

		if name == "" || filePath == "" {
			return binarySavedMsg{err: fmt.Errorf("name and file path are required")}
		}

		// Сохраняем файл с шифрованием
		destPath, checksum, size, err := m.storageManager.FileManager.SaveFile(name, filePath)
		if err != nil {
			return binarySavedMsg{err: fmt.Errorf("failed to save file: %w", err)}
		}

		// Определяем MIME тип
		contentType := mime.TypeByExtension(filepath.Ext(filePath))
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		// Извлекаем только имя файла
		filename := filepath.Base(filePath)

		var metadata *string
		if m.metadataInput.Value() != "" {
			val := m.metadataInput.Value()
			metadata = &val
		}

		// Создаем запись в БД
		binary := storage.CreateBinaryWithEncryption(
			name, // используем name как GetID временно
			name,
			filename,
			destPath,
			size,
			contentType,
			checksum,
			metadata,
		)

		err = m.storageManager.Encrypted.SaveBinary(binary)
		_ = ctx

		return binarySavedMsg{err: err}
	}
}

func (m *BinariesViewModel) deleteBinary() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if m.selectedBinary == nil {
			return binaryDeletedMsg{err: fmt.Errorf("no file selected")}
		}

		// Удаляем файл с диска
		if err := m.storageManager.FileManager.DeleteFile(m.selectedBinary.FilePath); err != nil {
			return binaryDeletedMsg{err: fmt.Errorf("failed to delete file: %w", err)}
		}

		// Удаляем запись из БД
		err := m.storageManager.Encrypted.DeleteBinary(m.selectedBinary.ID)
		_ = ctx

		return binaryDeletedMsg{err: err}
	}
}

// ====================
// Helpers
// ====================

func formatFileSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d B", size)
	}
}
