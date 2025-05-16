package tui

// DescriptionEditorModal provides a modal textarea for editing descriptions.
import (
	"fmt"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

type errMsg error

// DescriptionEditorModal holds the textarea and error state for the modal.
type DescriptionEditorModal struct {
	textarea textarea.Model
	err      error
}

// initialModel creates a new DescriptionEditorModal with a focused textarea.
func initialModel() DescriptionEditorModal {
	ti := textarea.New()
	ti.Placeholder = "Once upon a time..."
	ti.Focus()

	return DescriptionEditorModal{
		textarea: ti,
		err:      nil,
	}
}

// Init returns the initial command for the modal (textarea blink).
func (m DescriptionEditorModal) Init() tea.Cmd {
	return textarea.Blink
}

// Update handles messages for the modal, including key events and errors.
func (m DescriptionEditorModal) Update(msg tea.Msg) (DescriptionEditorModal, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			if m.textarea.Focused() {
				m.textarea.Blur()
			}
		case tea.KeyCtrlC:
			return m, tea.Quit
		default:
			if !m.textarea.Focused() {
				cmd = m.textarea.Focus()
				cmds = append(cmds, cmd)
			}
		}
	case errMsg:
		m.err = msg
		return m, nil
	}

	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

// Description returns the current value of the textarea.
func (m DescriptionEditorModal) Description() string {
	return m.textarea.Value()
}

// View renders the modal UI.
func (m DescriptionEditorModal) View(title string) string {
	return fmt.Sprintf(
		"%s\n\n%s\n\n%s",
		editHeaderStyle.Render(title),
		m.textarea.View(),
		"(ctrl+s to save)",
	) + "\n\n"
}

func NewEditModal(initial string) DescriptionEditorModal {
	modal := initialModel()
	modal.textarea.SetValue(initial)
	return modal
}
