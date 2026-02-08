package models

import (
	"fmt"
	"strings"
	"time"
)

// Card represents a bank card entry in the domain layer.
type Card struct {
	ID             string
	UserID         string
	Name           string
	CardNumber     string
	CardholderName string
	ExpiryDate     string
	CVV            string
	BankName       *string
	Metadata       *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
}

func (c *Card) GetID() string {
	return c.ID
}

func (c *Card) SetID(id string) {
	c.ID = id
}

func (c *Card) GetUserID() string {
	return c.UserID
}

// NewCard creates a new card entry with validation.
func NewCard(userID, name, cardNumber, cardholderName, expiryDate, cvv string, bankName, metadata *string) (*Card, error) {
	if userID == "" {
		return nil, ErrInvalidUserID
	}
	if name == "" {
		return nil, ErrInvalidName
	}
	if cardNumber == "" {
		return nil, ErrInvalidCardNumber
	}
	if cardholderName == "" {
		return nil, ErrInvalidCardholderName
	}
	if expiryDate == "" {
		return nil, ErrInvalidExpiryDate
	}
	if cvv == "" {
		return nil, ErrInvalidCVV
	}

	now := time.Now()
	card := &Card{
		UserID:         userID,
		Name:           name,
		CardNumber:     cardNumber,
		CardholderName: cardholderName,
		ExpiryDate:     expiryDate,
		CVV:            cvv,
		BankName:       bankName,
		Metadata:       metadata,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	return card, nil
}

// UpdateCardNumber updates the card number and sets the updated timestamp.
func (c *Card) UpdateCardNumber(cardNumber string) error {
	if cardNumber == "" {
		return ErrInvalidCardNumber
	}
	c.CardNumber = cardNumber
	c.UpdatedAt = time.Now()
	return nil
}

// UpdateCardholderName updates the cardholder name and sets the updated timestamp.
func (c *Card) UpdateCardholderName(cardholderName string) error {
	if cardholderName == "" {
		return ErrInvalidCardholderName
	}
	c.CardholderName = cardholderName
	c.UpdatedAt = time.Now()
	return nil
}

// UpdateExpiryDate updates the expiry date and sets the updated timestamp.
func (c *Card) UpdateExpiryDate(expiryDate string) error {
	if expiryDate == "" {
		return ErrInvalidExpiryDate
	}
	c.ExpiryDate = expiryDate
	c.UpdatedAt = time.Now()
	return nil
}

// UpdateCVV updates the CVV and sets the updated timestamp.
func (c *Card) UpdateCVV(cvv string) error {
	if cvv == "" {
		return ErrInvalidCVV
	}
	c.CVV = cvv
	c.UpdatedAt = time.Now()
	return nil
}

// UpdateBankName updates the bank name and sets the updated timestamp.
func (c *Card) UpdateBankName(bankName *string) {
	c.BankName = bankName
	c.UpdatedAt = time.Now()
}

// UpdateMetadata updates the metadata and sets the updated timestamp.
func (c *Card) UpdateMetadata(metadata *string) {
	c.Metadata = metadata
	c.UpdatedAt = time.Now()
}

// GetMaskedCardNumber returns the card number with sensitive digits masked.
// Example: "1234567890123456" -> "**** **** **** 3456"
func (c *Card) GetMaskedCardNumber() string {
	return MaskCardNumber(c.CardNumber)
}

// MaskCardNumber masks a card number for display purposes.
// Example: "1234567890123456" -> "**** **** **** 3456"
func MaskCardNumber(cardNumber string) string {
	// Remove any spaces or dashes for processing
	cleanNumber := strings.ReplaceAll(strings.ReplaceAll(cardNumber, " ", ""), "-", "")

	if len(cleanNumber) < 4 {
		return cleanNumber
	}

	// Keep last 4 digits, mask the rest
	lastFour := cleanNumber[len(cleanNumber)-4:]
	masked := "**** **** **** " + lastFour

	return masked
}

// GetCardType attempts to determine the card type from the card number.
func (c *Card) GetCardType() string {
	cleanNumber := strings.ReplaceAll(strings.ReplaceAll(c.CardNumber, " ", ""), "-", "")

	if len(cleanNumber) < 4 {
		return "Unknown"
	}

	switch cleanNumber[0] {
	case '3':
		if len(cleanNumber) >= 2 && (cleanNumber[1] == '4' || cleanNumber[1] == '7') {
			return "American Express"
		}
	case '4':
		return "Visa"
	case '5':
		return "Mastercard"
	case '6':
		return "Discover"
	}

	return "Unknown"
}

// IsExpired checks if the card is expired based on the expiry date.
// Assumes expiry date format is "MM/YY"
func (c *Card) IsExpired() bool {
	if len(c.ExpiryDate) != 5 || c.ExpiryDate[2] != '/' {
		return false // Invalid format, can't determine
	}

	var month, year int
	n, err := fmt.Sscanf(c.ExpiryDate, "%02d/%02d", &month, &year)
	if err != nil || n != 2 {
		return false // Invalid format
	}

	// Convert to full year
	if year < 100 {
		year += 2000
	}

	now := time.Now()
	expiryYear := year
	expiryMonth := time.Month(month)

	// Card expires at the end of the expiry month
	if now.Year() > expiryYear || (now.Year() == expiryYear && now.Month() > expiryMonth) {
		return true
	}

	return false
}
