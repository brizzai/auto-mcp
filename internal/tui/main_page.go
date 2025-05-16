package tui

import (
	"fmt"
	"strings"

	"github.com/brizzai/auto-mcp/internal/parser"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MainPageKeyMap holds key bindings for the main page actions
type MainPageKeyMap struct {
	open key.Binding
	quit key.Binding
}

func newMainPageKeyMap() *MainPageKeyMap {
	return &MainPageKeyMap{
		open: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "Open Routes Editor"),
		),
		quit: key.NewBinding(
			key.WithKeys("ctrl+c", "q"),
			key.WithHelp("ctrl+c/q", "Quit"),
		),
	}
}

// MainPageModel represents the main landing page of the application
type MainPageModel struct {
	keys       *MainPageKeyMap
	width      int
	height     int
	routeTools []*parser.RouteTool
	selected   bool
}

// OpenListItemMsg is sent when the user chooses to open the list item modal
type OpenListItemMsg struct {
	RouteTools []*parser.RouteTool
}

// NewMainPageModel creates a new main page model
func NewMainPageModel(routeTools []*parser.RouteTool) MainPageModel {
	return MainPageModel{
		keys:       newMainPageKeyMap(),
		routeTools: routeTools,
		selected:   false,
	}
}

// Init initializes the model
func (m MainPageModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the main page
func (m MainPageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.open):
			return m, func() tea.Msg {
				return OpenListItemMsg{RouteTools: m.routeTools}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

// View renders the main page
func (m MainPageModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	title := titleStyle.Render("MCP API Routes Manager")

	descStyle := lipgloss.NewStyle().
		Padding(1, 0).
		Width(m.width - 4).
		Align(lipgloss.Center)

	description := descStyle.Render(
		"This application allows you to manage your MCP API routes.\n" +
			"You can view, filter, edit descriptions, and mark routes for removal.\n\n" +
			"The application currently manages " + pluralize(len(m.routeTools), "route") + ".",
	)

	// Route list preview style
	routePreviewStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#f56a96")).
		Padding(1, 1).
		Width(m.width - 10).
		Align(lipgloss.Left)

	// Build route preview content
	var routePreviewContent strings.Builder
	maxPreviewRoutes := 5
	displayedRoutes := len(m.routeTools)
	if displayedRoutes > maxPreviewRoutes {
		displayedRoutes = maxPreviewRoutes
	}

	for i := 0; i < displayedRoutes; i++ {
		route := m.routeTools[i]
		routePreviewContent.WriteString(fmt.Sprintf("%s %s\n",
			route.RouteConfig.Method,
			route.RouteConfig.Path,
		))
	}

	if len(m.routeTools) > maxPreviewRoutes {
		routePreviewContent.WriteString(fmt.Sprintf("\n... and %d more routes", len(m.routeTools)-maxPreviewRoutes))
	}

	routePreview := routePreviewStyle.Render(routePreviewContent.String())

	instructionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#f56a96")).
		Padding(1, 0).
		Width(m.width - 4).
		Align(lipgloss.Center)

	instruction := instructionStyle.Render("Press ENTER to open the routes editor")

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#626262", Dark: "#A49FA5"}).
		Width(m.width - 4).
		Align(lipgloss.Center)

	help := helpStyle.Render("Press q or Ctrl+C to quit")

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		"",
		title,
		"",
		description,
		"",
		routePreview,
		"",
		instruction,
		"",
		help,
	)

	return docStyle.Render(content)
}

// pluralize is a helper function that returns a string with the count and noun
// properly pluralized
func pluralize(count int, singular string) string {
	if count == 1 {
		return "1 " + singular
	}
	return strings.TrimSuffix(singular, "e") + "s"
}
