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

type cardViewMode int

const (
	cardViewList cardViewMode = iota
	cardViewAdd
	cardViewDetail
	cardViewEdit
	cardViewConfirmDelete
)

type cardItem struct {
	card *storage.LocalCard
}

func (i cardItem) FilterValue() string { return i.card.Name }
func (i cardItem) Title() string {
	statusIcon := getSyncStatusIcon(i.card.SyncStatus)
	masked := maskCardNumber(i.card.CardNumber)
	return fmt.Sprintf("%s %s (%s)", statusIcon, i.card.Name, masked)
}

func (i cardItem) Description() string {
	return fmt.Sprintf("Holder: %s | Exp: %s | Updated: %s",
		i.card.CardholderName,
		i.card.ExpiryDate,
		i.card.UpdatedAt.Format("2006-01-02 15:04"))
}

// CardsViewModel manages cards CRUD operations.
type CardsViewModel struct {
	mode           cardViewMode
	list           list.Model
	offlineService *service.OfflineService
	cards          []*storage.LocalCard
	selectedCard   *storage.LocalCard

	// Form inputs
	nameInput       textinput.Model
	cardNumberInput textinput.Model
	holderInput     textinput.Model
	expiryInput     textinput.Model
	cvvInput        textinput.Model
	bankInput       textinput.Model
	metadataInput   textinput.Model
	focusedField    int

	err      error
	message  string
	quitting bool
	loading  bool
}

// NewCardsViewModel creates a new cards view model.
func NewCardsViewModel(offlineService *service.OfflineService) CardsViewModel {
	items := []list.Item{}
	l := list.New(items, list.NewDefaultDelegate(), 80, 20)
	l.Title = "💳 Bank Cards"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = menuTitleStyle

	nameInput := textinput.New()
	nameInput.Placeholder = "Card name (e.g., Work Visa)"
	nameInput.Width = 50

	cardNumberInput := textinput.New()
	cardNumberInput.Placeholder = "Card number (16 digits)"
	cardNumberInput.Width = 50

	holderInput := textinput.New()
	holderInput.Placeholder = "Cardholder name"
	holderInput.Width = 50

	expiryInput := textinput.New()
	expiryInput.Placeholder = "Expiry date (MM/YY)"
	expiryInput.Width = 50

	cvvInput := textinput.New()
	cvvInput.Placeholder = "CVV (3-4 digits)"
	cvvInput.EchoMode = textinput.EchoPassword
	cvvInput.Width = 50

	bankInput := textinput.New()
	bankInput.Placeholder = "Bank name (optional)"
	bankInput.Width = 50

	metadataInput := textinput.New()
	metadataInput.Placeholder = "Metadata (optional, JSON)"
	metadataInput.Width = 50

	return CardsViewModel{
		mode:            cardViewList,
		list:            l,
		offlineService:  offlineService,
		nameInput:       nameInput,
		cardNumberInput: cardNumberInput,
		holderInput:     holderInput,
		expiryInput:     expiryInput,
		cvvInput:        cvvInput,
		bankInput:       bankInput,
		metadataInput:   metadataInput,
	}
}

func (m CardsViewModel) Init() tea.Cmd {
	return m.loadCards()
}

func (m CardsViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Если уже выходим, игнорируем все сообщения кроме KeyMsg
	if m.quitting {
		if _, ok := msg.(tea.KeyMsg); !ok {
			return m, nil
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
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
		case cardViewList:
			return m.handleListKeys(key)
		case cardViewAdd, cardViewEdit:
			return m.handleFormKeys(msg)
		case cardViewDetail:
			return m.handleDetailKeys(key)
		case cardViewConfirmDelete:
			return m.handleConfirmKeys(key)
		}

	case cardsLoadedMsg:
		m.loading = false
		m.cards = msg.cards
		m.err = msg.err

		if m.err == nil {
			items := make([]list.Item, len(m.cards))
			for i, card := range m.cards {
				items[i] = cardItem{card: card}
			}
			m.list.SetItems(items)
		}
		return m, nil

	case cardSavedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.message = "Card saved successfully!"
		m.mode = cardViewList
		return m, m.loadCards()

	case cardDeletedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.message = "Card deleted successfully!"
		m.mode = cardViewList
		return m, m.loadCards()
	}

	if m.mode == cardViewList {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	if m.mode == cardViewAdd || m.mode == cardViewEdit {
		return m.updateActiveInput(msg)
	}

	return m, nil
}

func (m CardsViewModel) View() string {
	if m.quitting {
		return ""
	}

	switch m.mode {
	case cardViewList:
		return m.viewList()
	case cardViewAdd:
		return m.viewForm("Add Bank Card")
	case cardViewEdit:
		return m.viewForm("Edit Bank Card")
	case cardViewDetail:
		return m.viewDetail()
	case cardViewConfirmDelete:
		return m.viewConfirmDelete()
	}

	return "Unknown view"
}

func (m CardsViewModel) viewList() string {
	if m.loading {
		return "Loading cards..."
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

func (m *CardsViewModel) handleListKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		// Возврат в главное меню
		m.quitting = true
		return m, tea.Quit

	case "a":
		m.mode = cardViewAdd
		m.clearForm()
		m.focusedField = 0
		m.err = nil
		m.message = ""
		return m, m.nameInput.Focus()

	case "e":
		if len(m.cards) == 0 {
			return m, nil
		}
		selectedItem := m.list.SelectedItem()
		if selectedItem == nil {
			return m, nil
		}
		item := selectedItem.(cardItem)
		m.selectedCard = item.card
		m.mode = cardViewEdit
		m.loadFormFromCard(m.selectedCard)
		m.focusedField = 0
		m.err = nil
		m.message = ""
		return m, m.nameInput.Focus()

	case "d":
		if len(m.cards) == 0 {
			return m, nil
		}
		selectedItem := m.list.SelectedItem()
		if selectedItem == nil {
			return m, nil
		}
		item := selectedItem.(cardItem)
		m.selectedCard = item.card
		m.mode = cardViewConfirmDelete
		return m, nil

	case "enter":
		if len(m.cards) == 0 {
			return m, nil
		}
		selectedItem := m.list.SelectedItem()
		if selectedItem == nil {
			return m, nil
		}
		item := selectedItem.(cardItem)
		m.selectedCard = item.card
		m.mode = cardViewDetail
		return m, nil
	}

	return m, nil
}

func (m CardsViewModel) viewForm(title string) string {
	activeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true)
	inactiveStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	fields := []string{
		m.renderField("Card Name:", m.nameInput, 0, activeStyle, inactiveStyle),
		m.renderField("Card Number:", m.cardNumberInput, 1, activeStyle, inactiveStyle),
		m.renderField("Cardholder Name:", m.holderInput, 2, activeStyle, inactiveStyle),
		m.renderField("Expiry Date:", m.expiryInput, 3, activeStyle, inactiveStyle),
		m.renderField("CVV:", m.cvvInput, 4, activeStyle, inactiveStyle),
		m.renderField("Bank:", m.bankInput, 5, activeStyle, inactiveStyle),
		m.renderField("Metadata:", m.metadataInput, 6, activeStyle, inactiveStyle),
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

	return lipgloss.Place(
		100,
		40,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

func (m CardsViewModel) renderField(label string, input textinput.Model, fieldIndex int, activeStyle, inactiveStyle lipgloss.Style) string {
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

func (m *CardsViewModel) handleFormKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "esc":
		m.mode = cardViewList
		m.clearForm()
		m.err = nil
		return m, nil

	case "tab", "shift+tab":
		m.err = nil
		if key == "tab" {
			m.focusedField = (m.focusedField + 1) % 7
		} else {
			m.focusedField = (m.focusedField - 1 + 7) % 7
		}
		return m, m.focusField()

	case "enter":
		return m, m.saveCard()
	}

	return m.updateActiveInput(msg)
}

func (m *CardsViewModel) focusField() tea.Cmd {
	inputs := []*textinput.Model{
		&m.nameInput,
		&m.cardNumberInput,
		&m.holderInput,
		&m.expiryInput,
		&m.cvvInput,
		&m.bankInput,
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

func (m *CardsViewModel) updateActiveInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.focusedField {
	case 0:
		m.nameInput, cmd = m.nameInput.Update(msg)
	case 1:
		m.cardNumberInput, cmd = m.cardNumberInput.Update(msg)
	case 2:
		m.holderInput, cmd = m.holderInput.Update(msg)
	case 3:
		m.expiryInput, cmd = m.expiryInput.Update(msg)
	case 4:
		m.cvvInput, cmd = m.cvvInput.Update(msg)
	case 5:
		m.bankInput, cmd = m.bankInput.Update(msg)
	case 6:
		m.metadataInput, cmd = m.metadataInput.Update(msg)
	}

	return m, cmd
}

func (m *CardsViewModel) clearForm() {
	m.nameInput.SetValue("")
	m.cardNumberInput.SetValue("")
	m.holderInput.SetValue("")
	m.expiryInput.SetValue("")
	m.cvvInput.SetValue("")
	m.bankInput.SetValue("")
	m.metadataInput.SetValue("")
}

func (m *CardsViewModel) loadFormFromCard(card *storage.LocalCard) {
	m.nameInput.SetValue(card.Name)
	m.cardNumberInput.SetValue(card.CardNumber)
	m.holderInput.SetValue(card.CardholderName)
	m.expiryInput.SetValue(card.ExpiryDate)
	m.cvvInput.SetValue(card.CVV)

	if card.BankName != nil {
		m.bankInput.SetValue(*card.BankName)
	}
	if card.Metadata != nil {
		m.metadataInput.SetValue(*card.Metadata)
	}
}

func (m CardsViewModel) viewDetail() string {
	if m.selectedCard == nil {
		return "No card selected"
	}

	card := m.selectedCard

	details := []string{
		fmt.Sprintf("Name:           %s", card.Name),
		fmt.Sprintf("Card Number:    %s", card.CardNumber),
		fmt.Sprintf("Cardholder:     %s", card.CardholderName),
		fmt.Sprintf("Expiry Date:    %s", card.ExpiryDate),
		fmt.Sprintf("CVV:            %s", card.CVV),
	}

	if card.BankName != nil && *card.BankName != "" {
		details = append(details, fmt.Sprintf("Bank:           %s", *card.BankName))
	}

	if card.Metadata != nil && *card.Metadata != "" {
		details = append(details, fmt.Sprintf("Metadata:       %s", *card.Metadata))
	}

	details = append(details,
		"",
		fmt.Sprintf("Status:         %s", GetSyncStatusStyled(string(card.SyncStatus))),
		fmt.Sprintf("Created:        %s", card.CreatedAt.Format("2006-01-02 15:04:05")),
		fmt.Sprintf("Updated:        %s", card.UpdatedAt.Format("2006-01-02 15:04:05")),
	)

	if card.SyncedAt != nil {
		details = append(details, fmt.Sprintf("Synced:         %s", card.SyncedAt.Format("2006-01-02 15:04:05")))
	}

	detailsText := lipgloss.JoinVertical(lipgloss.Left, details...)
	help := menuHelpStyle.Render("\ne: edit | d: delete | Esc: back")

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		titleStyle.Render("💳 Card Details"),
		"",
		detailsText,
		help,
	)

	return lipgloss.Place(
		100,
		35,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

func (m *CardsViewModel) handleDetailKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		m.mode = cardViewList
		m.selectedCard = nil
		return m, nil

	case "e":
		m.mode = cardViewEdit
		m.loadFormFromCard(m.selectedCard)
		m.focusedField = 0
		return m, m.nameInput.Focus()

	case "d":
		m.mode = cardViewConfirmDelete
		return m, nil
	}

	return m, nil
}

func (m CardsViewModel) viewConfirmDelete() string {
	if m.selectedCard == nil {
		return "No card selected"
	}

	warning := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true).
		Render(fmt.Sprintf("⚠️  Delete card '%s'?", m.selectedCard.Name))

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

	return lipgloss.Place(
		80,
		20,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

func (m *CardsViewModel) handleConfirmKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "y":
		return m, m.deleteCard()
	case "n", "esc":
		m.mode = cardViewList
		m.selectedCard = nil
		return m, nil
	}
	return m, nil
}

type cardsLoadedMsg struct {
	cards []*storage.LocalCard
	err   error
}

type cardSavedMsg struct {
	err error
}

type cardDeletedMsg struct {
	err error
}

func (m CardsViewModel) loadCards() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cards, err := m.offlineService.ListCards(ctx)
		return cardsLoadedMsg{cards: cards, err: err}
	}
}

func (m *CardsViewModel) saveCard() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		name := m.nameInput.Value()
		cardNumber := m.cardNumberInput.Value()
		holder := m.holderInput.Value()
		expiry := m.expiryInput.Value()
		cvv := m.cvvInput.Value()

		if name == "" || cardNumber == "" || holder == "" || expiry == "" || cvv == "" {
			return cardSavedMsg{err: fmt.Errorf("name, card number, holder, expiry, and CVV are required")}
		}

		var bank, metadata *string
		if m.bankInput.Value() != "" {
			val := m.bankInput.Value()
			bank = &val
		}
		if m.metadataInput.Value() != "" {
			val := m.metadataInput.Value()
			metadata = &val
		}

		var err error
		if m.mode == cardViewAdd {
			_, err = m.offlineService.CreateCard(ctx, name, cardNumber, holder, expiry, cvv, bank, metadata)
		} else if m.mode == cardViewEdit && m.selectedCard != nil {
			_, err = m.offlineService.UpdateCard(ctx, m.selectedCard.ID, name, cardNumber, holder, expiry, cvv, bank, metadata)
		}

		return cardSavedMsg{err: err}
	}
}

func (m *CardsViewModel) deleteCard() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if m.selectedCard == nil {
			return cardDeletedMsg{err: fmt.Errorf("no card selected")}
		}

		err := m.offlineService.DeleteCard(ctx, m.selectedCard.ID)
		return cardDeletedMsg{err: err}
	}
}

func maskCardNumber(number string) string {
	if len(number) <= 4 {
		return "****"
	}
	return "**** **** **** " + number[len(number)-4:]
}
