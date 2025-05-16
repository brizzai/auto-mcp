package tui

import (
	"github.com/brizzai/auto-mcp/internal/tui/models"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// newItemDelegate returns a list.DefaultDelegate with custom update and help functions.
func newItemDelegate(keys *delegateKeyMap) list.DefaultDelegate {
	d := list.NewDefaultDelegate()

	d.UpdateFunc = func(msg tea.Msg, m *list.Model) tea.Cmd {
		var title string

		item, ok := m.SelectedItem().(models.RouteToolItem)
		if ok {
			title = item.Title()
		} else {
			return nil
		}

		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.remove):
				index := m.Index()
				updatedItem := item.ToggleRemoved()
				m.SetItem(index, updatedItem)
				if updatedItem.IsRemoved {
					return m.NewStatusMessage(statusMessageStyle("Removed " + title + " from the MCP list"))
				}
				return m.NewStatusMessage(statusMessageStyle("Added Back" + title + " to the MCP list"))
			}
		}
		return nil
	}

	help := []key.Binding{keys.remove}

	d.ShortHelpFunc = func() []key.Binding {
		return help
	}

	d.FullHelpFunc = func() [][]key.Binding {
		return [][]key.Binding{help}
	}

	return d
}

// delegateKeyMap holds key bindings for list item actions.
type delegateKeyMap struct {
	remove key.Binding
}

// ShortHelp returns additional short help entries for the delegate.
func (d delegateKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		d.remove,
	}
}

// FullHelp returns additional full help entries for the delegate.
func (d delegateKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{
			d.remove,
		},
	}
}

// newDelegateKeyMap creates a new delegateKeyMap with default bindings.
func newDelegateKeyMap() *delegateKeyMap {
	return &delegateKeyMap{
		remove: key.NewBinding(
			key.WithKeys("x", "backspace"),
			key.WithHelp("x", "Remove from MCP list"),
		),
	}
}
