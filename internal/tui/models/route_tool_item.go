package models

import (
	"fmt"

	"github.com/brizzai/auto-mcp/internal/parser"
	"github.com/charmbracelet/lipgloss"
)

// RouteToolItem wraps a RouteTool for display in the list
// Implements list.Item
type RouteToolItem struct {
	Tool           *parser.RouteTool
	NewDescription string
	IsRemoved      bool
}

func (i RouteToolItem) Title() string {
	return fmt.Sprintf("%s %s ", i.Tool.RouteConfig.Method, i.Tool.RouteConfig.Path)
}

func (i RouteToolItem) Description() string {
	if i.IsRemoved {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Render("[Removed]")
	}
	if i.NewDescription != "" {
		return i.NewDescription
	}
	return i.Tool.RouteConfig.Description
}

func (i RouteToolItem) UpdatedDescription(newDescription string) RouteToolItem {
	i.NewDescription = newDescription
	return i
}

func (i RouteToolItem) ToggleRemoved() RouteToolItem {
	i.IsRemoved = !i.IsRemoved
	return i
}

func (i RouteToolItem) FilterValue() string {
	return i.Tool.RouteConfig.Path + " " + i.Tool.RouteConfig.Description
}
