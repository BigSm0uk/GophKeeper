package tui

import "github.com/charmbracelet/lipgloss"

// Colors for sync statuses.
var (
	colorSynced    = lipgloss.Color("42")  // green
	colorPending   = lipgloss.Color("226") // yellow
	colorUploading = lipgloss.Color("39")  // blue
	colorUpdated   = lipgloss.Color("208") // orange
	colorDeleted   = lipgloss.Color("240") // gray
	colorError     = lipgloss.Color("196") // red
)

// Styles for sync statuses.
var (
	syncedStyle      = lipgloss.NewStyle().Foreground(colorSynced).Bold(true)
	pendingStyle     = lipgloss.NewStyle().Foreground(colorPending).Bold(true)
	uploadingStyle   = lipgloss.NewStyle().Foreground(colorUploading).Bold(true)
	updatedStyle     = lipgloss.NewStyle().Foreground(colorUpdated).Bold(true)
	deletedStyle     = lipgloss.NewStyle().Foreground(colorDeleted).Bold(true).Strikethrough(true)
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
	case "deleted":
		return deletedStyle.Render("🗑 deleted")
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
	case "deleted":
		return colorDeleted
	default:
		return colorError
	}
}
