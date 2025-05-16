package tui

import (
	"github.com/brizzai/auto-mcp/internal/parser"
	"github.com/brizzai/auto-mcp/internal/tui/models"
	tea "github.com/charmbracelet/bubbletea"
)

// AppModel is the main application model that manages page switching
type AppModel struct {
	mainPage   MainPageModel
	listView   ListItemModel
	exportView ExportView
	page       string // "main" or "list" or "export"
}

// NewAppModel creates a new AppModel with the provided route tools
func NewAppModel(routeTools []*parser.RouteTool, adjuster *parser.Adjuster) AppModel {
	return AppModel{
		mainPage:   NewMainPageModel(routeTools),
		listView:   NewListItemModel(routeTools, adjuster),
		exportView: ExportView{}, // Initialize with empty export view as we'll set it properly in DoneMsg
		page:       "main",
	}
}

// Init initializes the AppModel
func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.mainPage.Init(),
		m.listView.Init(),
	)
}

// Update handles app-level messages and delegates to the appropriate page model
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case OpenListItemMsg:
		m.page = "list"
		cmd := m.listView.Init()
		return m, cmd

	case DoneMsg:
		m.page = "export"
		m.exportView = NewExportView(m.listView.GetRoutesUpdates())
		cmd := m.exportView.Init()
		return m, cmd

	case BackToMainMsg:
		m.page = "list"
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "esc" && m.page == "list" {
			m.page = "main"
			return m, nil
		}

	case tea.WindowSizeMsg:
		var cmd tea.Cmd
		var tempModel tea.Model

		// Update all models with the window size
		tempModel, cmd = m.mainPage.Update(msg)
		m.mainPage = tempModel.(MainPageModel)
		cmds = append(cmds, cmd)

		tempModel, cmd = m.listView.Update(msg)
		m.listView = tempModel.(ListItemModel)
		cmds = append(cmds, cmd)

		tempModel, cmd = m.exportView.Update(msg)
		m.exportView = tempModel.(ExportView)
		cmds = append(cmds, cmd)

		return m, tea.Batch(cmds...)
	}

	// Delegate message to the active page
	var cmd tea.Cmd
	var tempModel tea.Model
	switch m.page {
	case "main":
		tempModel, cmd = m.mainPage.Update(msg)
		m.mainPage = tempModel.(MainPageModel)
		cmds = append(cmds, cmd)
	case "list":
		tempModel, cmd = m.listView.Update(msg)
		m.listView = tempModel.(ListItemModel)
		cmds = append(cmds, cmd)
	case "export":
		tempModel, cmd = m.exportView.Update(msg)
		m.exportView = tempModel.(ExportView)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the active page
func (m AppModel) View() string {
	switch m.page {
	case "main":
		return m.mainPage.View()
	case "export":
		return m.exportView.View()
	default: // list
		return m.listView.View()
	}
}

// GetFilteredRoutes delegates to the list view
func (m AppModel) GetRoutesUpdates() []*models.RouteToolItem {
	return m.listView.GetRoutesUpdates()
}

// IsFinished checks if the user has completed the TUI flow
// by verifying they've reached the export page
func (m AppModel) IsFinished() bool {
	return m.exportView.Success
}
