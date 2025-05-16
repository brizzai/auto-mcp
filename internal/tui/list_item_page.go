package tui

import (
	"github.com/brizzai/auto-mcp/internal/parser"
	"github.com/brizzai/auto-mcp/internal/tui/models"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"

	tea "github.com/charmbracelet/bubbletea"
)

// listKeyMap holds key bindings for the list actions.
type listKeyMap struct {
	editDescription key.Binding
	save            key.Binding
	finish          key.Binding
	quit            key.Binding
}

type DoneMsg struct {
	RouteTools []*models.RouteToolItem
}

// newListKeyMap creates a new listKeyMap with default bindings.
func newListKeyMap() *listKeyMap {
	return &listKeyMap{
		editDescription: key.NewBinding(
			key.WithKeys("E", "e"),
			key.WithHelp("E", "Edit Description"),
		),
		save: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "Save"),
		),
		finish: key.NewBinding(
			key.WithKeys("F", "f"),
			key.WithHelp("F", "Finish"),
		),
		quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "Quit"),
		),
	}
}

// ListItemModel for the TUI
type ListItemModel struct {
	list      list.Model
	keys      *listKeyMap
	editing   bool
	editIndex int
	editModal DescriptionEditorModal // Holds the edit modal when editing
}

// Init returns the initial command for the list model.
func (m ListItemModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the list and modal, including editing logic.
func (m ListItemModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.editing {
		return m.handleEditModeUpdate(msg)
	}
	return m.handleListModeUpdate(msg)
}

// handleEditModeUpdate handles messages when in edit mode
func (m ListItemModel) handleEditModeUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, m.keys.save) {
			m.editing = false
			item := m.list.SelectedItem().(models.RouteToolItem)
			newDescription := m.editModal.Description()
			if newDescription != item.Tool.RouteConfig.Description {
				m.list.SetItem(m.editIndex, item.UpdatedDescription(newDescription))
				m.list.NewStatusMessage(statusMessageStyle("Updated description for", m.list.SelectedItem().(models.RouteToolItem).Title()))
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}
	var cmd tea.Cmd
	m.editModal, cmd = m.editModal.Update(msg)
	return m, cmd
}

// handleListModeUpdate handles messages when in list mode
func (m ListItemModel) handleListModeUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.editDescription):
			idx := m.list.Index()
			item, ok := m.list.SelectedItem().(models.RouteToolItem)
			if ok {
				if item.IsRemoved {
					m.list.NewStatusMessage(statusMessageStyle("Can't edit removed routes", ""))
					return m, nil
				}
				m.editing = true
				m.editIndex = idx
				// Create the modal with the current description as initial value
				m.editModal = NewEditModal(item.Description())
				return m, nil
			}
		case key.Matches(msg, m.keys.finish):
			return m, func() tea.Msg {
				return DoneMsg{RouteTools: m.GetRoutesUpdates()}
			}
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders either the list or the modal
func (m ListItemModel) View() string {
	if m.editing {
		return docStyle.Render(m.editModal.View(m.list.SelectedItem().(models.RouteToolItem).Title()))
	}
	return docStyle.Render(m.list.View())
}

// NewModel creates a TUI model for a list of RouteTool
func NewListItemModel(routeTools []*parser.RouteTool, adjuster *parser.Adjuster) ListItemModel {
	listKeys := newListKeyMap()

	items := make([]list.Item, len(routeTools))
	for i, rt := range routeTools {
		items[i] = models.RouteToolItem{
			Tool:           rt,
			NewDescription: adjuster.GetDescription(rt.RouteConfig.Path, rt.RouteConfig.Method, ""),
			IsRemoved:      !adjuster.ExistsInMCP(rt.RouteConfig.Path, rt.RouteConfig.Method),
		}
	}
	delegateKeyMap := newDelegateKeyMap()
	delegate := newItemDelegate(delegateKeyMap)

	l := list.New(items, delegate, 0, 0)

	l.Title = titleStyle.Render("MCP API Routes editor")
	l.SetShowFilter(true)

	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			listKeys.editDescription,
			listKeys.finish,
			listKeys.quit,
		}
	}
	return ListItemModel{list: l, keys: listKeys, editing: false, editIndex: -1, editModal: DescriptionEditorModal{}}
}

// GetFilteredRoutes returns the currently visible (filtered) RouteTools
func (m ListItemModel) GetFilteredRoutes() []*parser.RouteTool {
	visible := m.list.VisibleItems()
	result := make([]*parser.RouteTool, len(visible))
	for i, item := range visible {
		result[i] = item.(models.RouteToolItem).Tool
	}
	return result
}

// GetRoutesUpdates returns the currently visible RouteToolItems with their updates
func (m ListItemModel) GetRoutesUpdates() []*models.RouteToolItem {
	visible := m.list.VisibleItems()
	result := make([]*models.RouteToolItem, len(visible))
	for i, item := range visible {
		// Create a pointer to a new RouteToolItem
		routeToolItem := item.(models.RouteToolItem)
		result[i] = &routeToolItem
	}
	return result
}
