package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	adjustments "github.com/brizzai/auto-mcp/internal/models"
	"github.com/brizzai/auto-mcp/internal/tui/models"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"gopkg.in/yaml.v3"
)

// ExportView handles prompting for a filename and exporting routes
type ExportView struct {
	routeTools   []*models.RouteToolItem
	textInput    textinput.Model
	err          error
	width        int
	height       int
	exportStatus string
	Success      bool
}

// NewExportView creates a new export view
func NewExportView(routeTools []*models.RouteToolItem) ExportView {
	ti := textinput.New()
	ti.Placeholder = "filename.yaml"
	ti.Focus()
	ti.Width = 40

	return ExportView{
		routeTools: routeTools,
		textInput:  ti,
	}
}

// Init initializes the export view
func (m ExportView) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages for the export view
func (m ExportView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			// Return to main page
			return m, func() tea.Msg { return BackToMainMsg{} }
		case "enter":
			// Process export
			if m.textInput.Value() == "" {
				m.exportStatus = "Please enter a filename"
				return m, nil
			}

			filename := m.textInput.Value()
			if !strings.HasSuffix(filename, ".yaml") && !strings.HasSuffix(filename, ".yml") {
				filename += ".yaml"
			}

			err := ExportRoutesToYamlFile(m.routeTools, filename)
			if err != nil {
				m.err = err
				m.exportStatus = fmt.Sprintf("Error exporting: %v", err)
				return m, nil
			}

			if _, err := os.Stat(filename); os.IsNotExist(err) {
				m.exportStatus = fmt.Sprintf("Error: File %s was not created", filename)
				return m, nil
			}

			m.Success = true
			m.exportStatus = completeMessageStyle(fmt.Sprintf("Successfully exported to %s", filename))
			// Wait for 1 second, then exit the application
			return m, tea.Sequence(
				tea.Tick(time.Second*1, func(time.Time) tea.Msg {
					return tea.Quit()
				}),
			)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// View renders the export view
func (m ExportView) View() string {
	var sb strings.Builder

	// Calculate vertical centering
	verticalPadding := (m.height - 6) / 2
	for i := 0; i < verticalPadding; i++ {
		sb.WriteString("\n")
	}

	title := titleStyle.Render("Export Routes")
	sb.WriteString(centerText(title, m.width))
	sb.WriteString("\n\n")

	prompt := "Enter filename to export routes:"
	sb.WriteString(centerText(prompt, m.width))
	sb.WriteString("\n")

	input := m.textInput.View()
	sb.WriteString(centerText(input, m.width))
	sb.WriteString("\n\n")

	if m.exportStatus != "" {
		sb.WriteString(centerText(m.exportStatus, m.width))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(centerText("(esc) Back to main | (enter) Export", m.width))

	return sb.String()
}

// BackToMainMsg signals to go back to the main page
type BackToMainMsg struct{}

// Helper function to export routes to a YAML file
func ExportRoutesToYamlFile(routes []*models.RouteToolItem, filename string) error {
	// Create the structure for YAML output
	exportData := adjustments.MCPAdjustments{
		Descriptions: []adjustments.RouteDescription{},
		Routes:       []adjustments.RouteSelection{},
	}

	// Group routes by path for both descriptions and selections
	descriptionsByPath := make(map[string][]adjustments.RouteFieldUpdate)
	methodsByPath := make(map[string][]string)

	// Process each route
	for _, route := range routes {
		path := route.Tool.RouteConfig.Path
		method := route.Tool.RouteConfig.Method

		// If route has a new description, add to descriptions
		if route.NewDescription != "" {
			descriptionsByPath[path] = append(descriptionsByPath[path], adjustments.RouteFieldUpdate{
				Method:         method,
				NewDescription: route.NewDescription,
			})
		}

		// If route is NOT marked as removed, add to routes
		if !route.IsRemoved {
			methodsByPath[path] = append(methodsByPath[path], method)
		}
	}

	// Convert grouped descriptions to RouteDescription slice
	for path, updates := range descriptionsByPath {
		exportData.Descriptions = append(exportData.Descriptions, adjustments.RouteDescription{
			Path:    path,
			Updates: updates,
		})
	}

	// Convert grouped methods to RouteSelection slice
	for path, methods := range methodsByPath {
		exportData.Routes = append(exportData.Routes, adjustments.RouteSelection{
			Path:    path,
			Methods: methods,
		})
	}

	// Convert to YAML
	yamlData, err := yaml.Marshal(exportData)
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(filename, yamlData, 0o644)
}

// Helper function to center text horizontally
func centerText(text string, width int) string {
	if width <= len(text) {
		return text
	}

	padding := (width - len(text)) / 2
	return strings.Repeat(" ", padding) + text
}
