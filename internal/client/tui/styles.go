package tui

import "github.com/charmbracelet/lipgloss"

// Цвета для статусов синхронизации
var (
	colorSynced    = lipgloss.Color("42")  // зеленый
	colorPending   = lipgloss.Color("226") // желтый
	colorUploading = lipgloss.Color("39")  // синий
	colorUpdated   = lipgloss.Color("208") // оранжевый
	colorError     = lipgloss.Color("196") // красный
)

// Стили для статусов
var (
	syncedStyle      = lipgloss.NewStyle().Foreground(colorSynced).Bold(true)
	pendingStyle     = lipgloss.NewStyle().Foreground(colorPending).Bold(true)
	uploadingStyle   = lipgloss.NewStyle().Foreground(colorUploading).Bold(true)
	updatedStyle     = lipgloss.NewStyle().Foreground(colorUpdated).Bold(true)
	errorStatusStyle = lipgloss.NewStyle().Foreground(colorError).Bold(true)
)

// GetSyncStatusStyled returns colored sync status text.
func GetSyncStatusStyled(status string) string {
	switch status {
	case "synced":
		return syncedStyle.Render("✓ synced")
	case "pending":
		return pendingStyle.Render("⏳ pending")
	case "uploading":
		return uploadingStyle.Render("📤 uploading")
	case "updated":
		return updatedStyle.Render("✎ updated")
	default:
		return errorStatusStyle.Render("? unknown")
	}
}

// GetSyncStatusColor returns color for sync status.
func GetSyncStatusColor(status string) lipgloss.Color {
	switch status {
	case "synced":
		return colorSynced
	case "pending":
		return colorPending
	case "uploading":
		return colorUploading
	case "updated":
		return colorUpdated
	default:
		return colorError
	}
}
