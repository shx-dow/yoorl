package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/shx-dow/yoorl/store"
	qrcode "github.com/skip2/go-qrcode"
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
		m.urlInput.Reset()
		m.aliasInput.Reset()
		return m, nil
		}

		switch m.screen {
		case screenList:
			return m.handleListKey(msg)
		case screenCreate:
			return m.handleCreateKey(msg)
		case screenConfirmDelete:
			return m.handleConfirmDeleteKey(msg)
		case screenQr:
			m.screen = screenList
			return m, nil
		}

	case urlsLoadedMsg:
		m.urls = []*store.UrlEntry(msg)
		m.err = nil
		m.clampCursor()
		if len(m.urls) > 0 {
			m.analytics = nil
			cmds = append(cmds, loadAnalytics(m.client, m.urls[m.cursor].ShortUrl))
		} else {
			m.analytics = nil
		}

	case analyticsLoadedMsg:
		m.analytics = msg
		m.err = nil

	case errMsg:
		m.err = error(msg)

	case tickMsg:
		cmds = append(cmds, loadURLs(m.client), tick())

	case noteTimeoutMsg:
		m.note = ""
	}

	return m, tea.Batch(cmds...)
}

func (m model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		if len(m.urls) > 0 {
			m.analytics = nil
			return m, loadAnalytics(m.client, m.urls[m.cursor].ShortUrl)
		}

	case "down", "j":
		if m.cursor < len(m.urls)-1 {
			m.cursor++
		}
		if len(m.urls) > 0 {
			m.analytics = nil
			return m, loadAnalytics(m.client, m.urls[m.cursor].ShortUrl)
		}

	case "enter":
		if len(m.urls) == 0 {
			return m, nil
		}
		m.analytics = nil
		return m, loadAnalytics(m.client, m.urls[m.cursor].ShortUrl)

	case "n":
		m.screen = screenCreate
		m.urlInput.Reset()
		m.aliasInput.Reset()
		m.urlInput.Focus()
		m.aliasInput.Blur()
		m.focusedInput = 0
		return m, nil

	case "d":
		if len(m.urls) == 0 {
			return m, nil
		}
		m.screen = screenConfirmDelete
		return m, nil

	case "c":
		if len(m.urls) == 0 {
			return m, nil
		}
		fullURL := m.client.BaseURL + "/r/" + m.urls[m.cursor].ShortUrl
		if err := clipboard.WriteAll(fullURL); err != nil {
			m.err = fmt.Errorf("clipboard: %w", err)
		} else {
			m.setNote("Copied to clipboard!", 2*time.Second)
			return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
				return noteTimeoutMsg{}
			})
		}

	case "v":
		if len(m.urls) == 0 {
			return m, nil
		}
		shortURL := m.urls[m.cursor].ShortUrl
		m.qrURL = m.client.BaseURL + "/r/" + shortURL
		qr, err := qrcode.New(m.qrURL, qrcode.Medium)
		if err != nil {
			m.err = fmt.Errorf("qr: %w", err)
			return m, nil
		}
		m.qrDisplay = qr.ToSmallString(false)
		m.screen = screenQr
		return m, nil

	case "r":
		return m, loadURLs(m.client)

	case "g":
		if len(m.urls) > 0 {
			m.cursor = 0
			m.analytics = nil
			return m, loadAnalytics(m.client, m.urls[0].ShortUrl)
		}

	case "G":
		if len(m.urls) > 0 {
			m.cursor = len(m.urls) - 1
			m.analytics = nil
			return m, loadAnalytics(m.client, m.urls[m.cursor].ShortUrl)
		}
	}

	return m, nil
}

func (m model) handleCreateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		if m.focusedInput == 0 {
			m.urlInput.Blur()
			m.aliasInput.Focus()
			m.focusedInput = 1
		} else {
			m.aliasInput.Blur()
			m.urlInput.Focus()
			m.focusedInput = 0
		}
		return m, nil

	case "enter":
		url := strings.TrimSpace(m.urlInput.Value())
		if url == "" {
			return m, nil
		}
		alias := strings.TrimSpace(m.aliasInput.Value())
		m.urlInput.Reset()
		m.aliasInput.Reset()
		m.screen = screenList
		return m, tea.Sequence(
			func() tea.Msg {
				if _, err := m.client.CreateURL(url, alias); err != nil {
					return errMsg(err)
				}
				return nil
			},
			loadURLs(m.client),
		)

	case "esc":
		m.screen = screenList
		m.urlInput.Reset()
		m.aliasInput.Reset()
		return m, nil

	default:
		var cmd tea.Cmd
		if m.focusedInput == 0 {
			m.urlInput, cmd = m.urlInput.Update(msg)
		} else {
			m.aliasInput, cmd = m.aliasInput.Update(msg)
		}
		return m, cmd
	}
}

func (m model) handleConfirmDeleteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if m.cursor < 0 || m.cursor >= len(m.urls) {
			m.screen = screenList
			return m, nil
		}
		shortURL := m.urls[m.cursor].ShortUrl
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

func (m *model) clampCursor() {
	if m.cursor >= len(m.urls) {
		m.cursor = len(m.urls) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
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
	case screenQr:
		b.WriteString(m.renderQr())
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

	if len(m.urls) == 0 {
		b.WriteString(helpStyle.Render(" No URLs yet. Press n to create one."))
		b.WriteString("\n")
		return b.String()
	}

	for i, u := range m.urls {
		dest := u.LongUrl
		maxDest := m.width - 50
		if maxDest < 20 {
			maxDest = 20
		}
		if len(dest) > maxDest {
			dest = dest[:maxDest-3] + "..."
		}

		fullShort := m.client.BaseURL + "/r/" + u.ShortUrl
		maxFull := m.width - len(u.ShortUrl) - maxDest - 10
		if maxFull < 10 {
			maxFull = 10
		}
		if len(fullShort) > maxFull {
			fullShort = fullShort[:maxFull-3] + "..."
		}

		line := fmt.Sprintf(" %d. %s  %s  %s",
			i+1,
			u.ShortUrl,
			dest,
			fullShort,
		)

		if i == m.cursor {
			b.WriteString(selectedStyle.Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")

	if m.note != "" {
		b.WriteString(noteStyle.Render(" " + m.note))
		b.WriteString("\n\n")
	}

	if m.err != nil {
		b.WriteString(errStyle.Render(fmt.Sprintf(" Error: %v", m.err)))
		b.WriteString("\n\n")
	}

	if m.analytics != nil {
		b.WriteString(detailStyle.Render(m.renderAnalyticsDetail()))
	}

	return b.String()
}

func (m model) renderAnalyticsDetail() string {
	a := m.analytics

	separator := strings.Repeat("─", m.width-4)
	if separator == "" {
		separator = "────────────────"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(" %s\n", separator))
	sb.WriteString(titleStyle.Render(fmt.Sprintf(" %s", a.ShortUrl)))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf(" Full URL: %s\n", m.client.BaseURL+"/r/"+a.ShortUrl))
	sb.WriteString(fmt.Sprintf(" Total clicks: %d\n\n", a.TotalClicks))

	if a.TotalClicks > 0 {
		sb.WriteString(" Recent visits:\n")
		maxShow := 5
		if len(a.RecentClicks) < maxShow {
			maxShow = len(a.RecentClicks)
		}
		for _, c := range a.RecentClicks[:maxShow] {
			ua := c.UserAgent
			if len(ua) > 40 {
				ua = ua[:40] + "..."
			}
			ts := c.Timestamp.Format("2006-01-02 15:04:05")
			sb.WriteString(fmt.Sprintf("   %s | %-15s | %s\n", ts, c.IP, ua))
		}
	}

	return sb.String()
}

func (m model) renderCreate() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Create new short URL"))
	b.WriteString("\n\n")

	urlLabel := " Long URL:"
	if m.focusedInput == 0 {
		urlLabel = cursorStyle.Render("▸ Long URL:")
	}
	b.WriteString(" " + urlLabel + "\n\n")
	b.WriteString("  " + m.urlInput.View() + "\n\n")

	aliasLabel := " Custom alias (optional):"
	if m.focusedInput == 1 {
		aliasLabel = cursorStyle.Render("▸ Custom alias (optional):")
	}
	b.WriteString(" " + aliasLabel + "\n\n")
	b.WriteString("  " + m.aliasInput.View() + "\n\n")

	b.WriteString(helpStyle.Render(" [tab] switch  [enter] create  [esc] cancel"))
	return b.String()
}

func (m model) renderConfirmDelete() string {
	if m.cursor < 0 || m.cursor >= len(m.urls) {
		return ""
	}
	u := m.urls[m.cursor]
	var b strings.Builder
	b.WriteString(errStyle.Render(" Delete confirmation"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf(" Delete %s → %s?\n\n", u.ShortUrl, u.LongUrl))
	b.WriteString(helpStyle.Render(" [y] yes  [any other key] cancel"))
	return b.String()
}

func (m model) renderQr() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" QR Code"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf(" %s\n\n", m.qrURL))
	b.WriteString(" " + strings.ReplaceAll(m.qrDisplay, "\n", "\n ") + "\n\n")
	b.WriteString(helpStyle.Render(" [esc] back"))
	return b.String()
}

func (m model) renderHelp() string {
	if m.screen == screenList {
		return helpStyle.Render(" [↑/↓] navigate  [enter] analytics  [n] new  [d] delete  [c] copy  [v] qr  [r] refresh  [q] quit")
	}
	return ""
}
