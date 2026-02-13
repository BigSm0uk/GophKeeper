package models

import (
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
