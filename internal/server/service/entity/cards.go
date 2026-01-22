package entity

import (
	"fmt"
	"strings"
	"time"

	pb "github.com/BigSm0uk/GophKeeper/pkg/proto/gophkeeper/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
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
}

// MapCardToResponse converts domain Card to protobuf Card message.
func MapCardToResponse(card *Card) (*pb.Card, error) {
	if card == nil {
		return nil, fmt.Errorf("card is nil")
	}

	response := &pb.Card{
		Id:             card.ID,
		Name:           card.Name,
		CardNumber:     card.CardNumber,
		CardholderName: card.CardholderName,
		ExpiryDate:     card.ExpiryDate,
		Cvv:            card.CVV,
		BankName:       card.BankName,
		Metadata:       card.Metadata,
		CreatedAt:      timestamppb.New(card.CreatedAt),
		UpdatedAt:      timestamppb.New(card.UpdatedAt),
	}

	return response, nil
}

// MapCardFromRequest converts protobuf CardCreateRequest to domain Card.
func MapCardFromRequest(userID string, req *pb.CardCreateRequest) (*Card, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}

	now := time.Now()
	card := &Card{
		UserID:         userID,
		Name:           req.Name,
		CardNumber:     req.CardNumber,
		CardholderName: req.CardholderName,
		ExpiryDate:     req.ExpiryDate,
		CVV:            req.Cvv,
		BankName:       req.BankName,
		Metadata:       req.Metadata,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	return card, nil
}

// MapCardFromUpdateRequest updates domain Card from protobuf CardUpdateRequest.
func MapCardFromUpdateRequest(existing *Card, req *pb.CardUpdateRequest) error {
	if existing == nil {
		return fmt.Errorf("existing card is nil")
	}
	if req == nil {
		return fmt.Errorf("request is nil")
	}

	existing.Name = req.Name
	existing.CardNumber = req.CardNumber
	existing.CardholderName = req.CardholderName
	existing.ExpiryDate = req.ExpiryDate
	existing.CVV = req.Cvv
	existing.BankName = req.BankName
	existing.Metadata = req.Metadata
	existing.UpdatedAt = time.Now()

	return nil
}

// MapCardToListItem converts domain Card to protobuf Card for list responses.
// This version includes masked card number for security.
func MapCardToListItem(card *Card) (*pb.Card, error) {
	if card == nil {
		return nil, fmt.Errorf("card is nil")
	}

	// For list responses, we mask the card number
	maskedNumber := MaskCardNumber(card.CardNumber)

	response := &pb.Card{
		Id:             card.ID,
		Name:           card.Name,
		CardNumber:     maskedNumber,
		CardholderName: card.CardholderName,
		ExpiryDate:     card.ExpiryDate,
		Cvv:            card.CVV,
		BankName:       card.BankName,
		Metadata:       card.Metadata,
		CreatedAt:      timestamppb.New(card.CreatedAt),
		UpdatedAt:      timestamppb.New(card.UpdatedAt),
	}

	return response, nil
}

// MaskCardNumber masks a card number for display purposes.
// Example: "1234567890123456" -> "**** **** **** 3456"
func MaskCardNumber(cardNumber string) string {
	if len(cardNumber) < 4 {
		return cardNumber
	}

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
