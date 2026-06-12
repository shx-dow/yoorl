package tui

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/shx-dow/yoorl/store"
)

func StartTUI(baseURL, apiKey string) error {
	client := NewClient(baseURL, apiKey)
	m := initialModel(client)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetWidth(msg.Width - 4)

		detailHeight := msg.Height - m.table.Height() - 8
		if detailHeight < 5 {
			detailHeight = 5
		}
		m.table.SetHeight(msg.Height - detailHeight - 8)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.screen == screenList {
				m.quitting = true
				return m, tea.Quit
			}
		case "esc":
			m.screen = screenList
			m.err = nil
			m.textInput.Reset()
			return m, nil
		}

		switch m.screen {
		case screenList:
			return m.handleListKey(msg)
		case screenCreate:
			return m.handleCreateKey(msg)
		case screenConfirmDelete:
			return m.handleConfirmDeleteKey(msg)
		}

	case urlsLoadedMsg:
		m.urls = []*store.UrlEntry(msg)
		m.err = nil
		cmds = append(cmds, m.updateTable())

	case analyticsLoadedMsg:
		m.analytics = msg
		m.err = nil

	case errMsg:
		m.err = error(msg)

	case tickMsg:
		cmds = append(cmds, loadURLs(m.client), tick())
	}

	return m, tea.Batch(cmds...)
}

func (m model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if len(m.table.Rows()) == 0 {
			return m, nil
		}
		row := m.table.SelectedRow()
		if len(row) < 2 {
			return m, nil
		}
		shortURL := row[1]
		return m, loadAnalytics(m.client, shortURL)

	case "n":
		m.screen = screenCreate
		m.textInput.Reset()
		m.textInput.Focus()
		return m, nil

	case "d":
		if len(m.table.Rows()) == 0 {
			return m, nil
		}
		m.screen = screenConfirmDelete
		return m, nil

	case "c":
		if len(m.table.Rows()) == 0 {
			return m, nil
		}
		row := m.table.SelectedRow()
		if len(row) < 2 {
			return m, nil
		}
		shortURL := row[1]
		if err := clipboard.WriteAll(m.client.BaseURL + "/r/" + shortURL); err != nil {
			m.err = fmt.Errorf("clipboard: %w", err)
		}
		return m, nil

	case "r":
		return m, loadURLs(m.client)

	default:
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) handleCreateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		url := strings.TrimSpace(m.textInput.Value())
		if url == "" {
			return m, nil
		}
		m.textInput.Reset()
		m.screen = screenList
		return m, tea.Sequence(
			func() tea.Msg {
				if _, err := m.client.CreateURL(url, ""); err != nil {
					return errMsg(err)
				}
				return nil
			},
			loadURLs(m.client),
		)

	case "esc":
		m.screen = screenList
		m.textInput.Reset()
		return m, nil

	default:
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}
}

func (m model) handleConfirmDeleteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		row := m.table.SelectedRow()
		if len(row) < 2 {
			m.screen = screenList
			return m, nil
		}
		shortURL := row[1]
		m.screen = screenList
		return m, tea.Sequence(
			func() tea.Msg {
				if err := m.client.DeleteURL(shortURL); err != nil {
					return errMsg(err)
				}
				return nil
			},
			loadURLs(m.client),
		)

	default:
		m.screen = screenList
		return m, nil
	}
}

func (m model) updateTable() tea.Cmd {
	rows := make([]table.Row, 0, len(m.urls))
	for i, u := range m.urls {
		dest := u.LongUrl
		if len(dest) > 47 {
			dest = dest[:47] + "..."
		}
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", i+1),
			u.ShortUrl,
			dest,
			fmt.Sprintf("%d", u.TotalClicks),
		})
	}
	m.table.SetRows(rows)

	if len(rows) > 0 {
		m.analytics = nil
		return loadAnalytics(m.client, rows[0][1])
	}
	return nil
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	b.WriteString(headerStyle.Render(fmt.Sprintf(" yoorl TUI  (%s)", m.client.BaseURL)))
	b.WriteString("\n\n")

	switch m.screen {
	case screenCreate:
		b.WriteString(m.renderCreate())
	case screenConfirmDelete:
		b.WriteString(m.renderConfirmDelete())
	default:
		b.WriteString(m.renderList())
	}

	b.WriteString("\n")
	b.WriteString(m.renderHelp())
	b.WriteString("\n")

	return b.String()
}

func (m model) renderList() string {
	var b strings.Builder

	b.WriteString(m.table.View())
	b.WriteString("\n")

	if m.err != nil {
		b.WriteString(errStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n")
	}

	if m.analytics != nil {
		b.WriteString(detailStyle.Render(m.renderAnalyticsDetail()))
	}

	return b.String()
}

func (m model) renderAnalyticsDetail() string {
	a := m.analytics

	var sb strings.Builder
	sb.WriteString(titleStyle.Render(fmt.Sprintf(" %s", a.ShortUrl)))
	sb.WriteString(fmt.Sprintf(" → %s\n", m.client.BaseURL+"/r/"+a.ShortUrl))
	sb.WriteString(fmt.Sprintf(" Total clicks: %d\n", a.TotalClicks))

	if len(a.RecentClicks) > 0 {
		sb.WriteString(" Recent visits:\n")
		maxShow := 4
		if len(a.RecentClicks) < maxShow {
			maxShow = len(a.RecentClicks)
		}
		for _, c := range a.RecentClicks[:maxShow] {
			ua := c.UserAgent
			if len(ua) > 30 {
				ua = ua[:30] + "..."
			}
			ts := c.Timestamp.Format("15:04:05")
			sb.WriteString(fmt.Sprintf("   %s | %s | %s\n", ts, c.IP, ua))
		}
	}

	return sb.String()
}

func (m model) renderCreate() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Create new short URL"))
	b.WriteString("\n\n")
	b.WriteString(" Enter the long URL:\n\n")
	b.WriteString(" " + m.textInput.View() + "\n\n")
	b.WriteString(helpStyle.Render(" [enter] create  [esc] cancel"))
	return b.String()
}

func (m model) renderConfirmDelete() string {
	row := m.table.SelectedRow()
	if len(row) < 2 {
		return ""
	}
	var b strings.Builder
	b.WriteString(errStyle.Render(" Delete confirmation"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf(" Delete %s → %s?\n\n", row[1], row[2]))
	b.WriteString(helpStyle.Render(" [y] yes  [any other key] cancel"))
	return b.String()
}

func (m model) renderHelp() string {
	if m.screen == screenList {
		return helpStyle.Render(" [n] new  [d] delete  [c] copy  [r] refresh  [q] quit")
	}
	return ""
}
