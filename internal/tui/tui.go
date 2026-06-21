package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
			m.foldOpen = false
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
		cmds = append(cmds, checkHealth(m.client), loadURLs(m.client), tick())

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
		m.foldOpen = false
		if len(m.urls) > 0 {
			m.analytics = nil
			return m, loadAnalytics(m.client, m.urls[m.cursor].ShortUrl)
		}

	case "down", "j":
		if m.cursor < len(m.urls)-1 {
			m.cursor++
		}
		m.foldOpen = false
		if len(m.urls) > 0 {
			m.analytics = nil
			return m, loadAnalytics(m.client, m.urls[m.cursor].ShortUrl)
		}

	case "enter":
		if len(m.urls) == 0 {
			return m, nil
		}
		m.foldOpen = !m.foldOpen
		if m.foldOpen && m.analytics == nil {
			return m, loadAnalytics(m.client, m.urls[m.cursor].ShortUrl)
		}
		return m, nil

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
			m.foldOpen = false
			m.analytics = nil
			return m, loadAnalytics(m.client, m.urls[0].ShortUrl)
		}

	case "G":
		if len(m.urls) > 0 {
			m.cursor = len(m.urls) - 1
			m.foldOpen = false
			m.analytics = nil
			return m, loadAnalytics(m.client, m.urls[m.cursor].ShortUrl)
		}
	case "?":
		return m, nil
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

	b.WriteString(m.renderStatsBar())
	b.WriteString("\n")

	switch m.screen {
	case screenCreate, screenConfirmDelete, screenQr:
		b.WriteString(m.renderOverlay())
	default:
		b.WriteString(m.renderList())
	}

	b.WriteString("\n")
	b.WriteString(m.renderHelp())
	b.WriteString("\n")

	return b.String()
}

func (m model) renderStatsBar() string {
	totalClicks := int64(0)
	for _, u := range m.urls {
		totalClicks += u.TotalClicks
	}

	left := statsBarStyle.Render(" YOORL")
	location := mutedStyle.Render("  links")
	status := noteStyle.Render("● connected")
	if m.err != nil {
		status = errStyle.Render("! disconnected")
	}
	right := fmt.Sprintf("%d URLs  %d clicks  %s", len(m.urls), totalClicks, status)
	width := m.width
	if width <= 0 {
		width = 100
	}
	gap := width - lipgloss.Width(left) - lipgloss.Width(location) - lipgloss.Width(right) - 2
	if gap < 2 {
		return left + location
	}
	return left + location + strings.Repeat(" ", gap) + mutedStyle.Render(right)
}

func (m model) renderList() string {
	if len(m.urls) == 0 {
		return "\n " + titleStyle.Render("No links yet.") + "\n " +
			mutedStyle.Render("Create a short URL to start tracking visits.") + "\n\n " +
			noteStyle.Render("n create first link") + "\n"
	}

	contentWidth := m.width
	if contentWidth <= 0 {
		contentWidth = 100
	}
	return m.renderCompactList(contentWidth)
}

func (m model) renderCompactList(width int) string {
	var b strings.Builder
	b.WriteString(m.renderTableHeader(width))
	b.WriteString("\n")
	for i, u := range m.urls {
		b.WriteString(m.renderURLRow(i, u, width))
		b.WriteString("\n")

		if i == m.cursor && m.foldOpen {
			if m.analytics != nil && m.analytics.ShortUrl == u.ShortUrl {
				b.WriteString(m.renderFold())
			} else if m.analytics == nil {
				b.WriteString("   Loading analytics...\n")
			}
		}
	}

	b.WriteString(m.renderMessages())
	return b.String()
}

func (m model) renderTableHeader(width int) string {
	if width < 70 {
		return dimStyle.Render(fmt.Sprintf(" %-3s %-16s %-*s %6s", "#", "ALIAS", maxInt(12, width-31), "DESTINATION", "CLICKS"))
	}
	return dimStyle.Render(fmt.Sprintf(" %-3s %-16s %-*s %8s", "#", "ALIAS", width-35, "DESTINATION", "CLICKS"))
}

func (m model) renderURLRow(i int, u *store.UrlEntry, width int) string {
	aliasWidth := 16
	destWidth := width - 35
	clickWidth := 8
	if width < 70 {
		aliasWidth = 16
		destWidth = maxInt(12, width-35)
		clickWidth = 6
	}
	alias := truncate(u.ShortUrl, aliasWidth)
	dest := truncate(u.LongUrl, destWidth)
	marker := " "
	if i == m.cursor {
		marker = "›"
	}
	line := fmt.Sprintf("%s%-3d %-*s %-*s %*d", marker, i+1, aliasWidth, alias, destWidth, dest, clickWidth, u.TotalClicks)
	if i == m.cursor {
		return selectedStyle.Width(width).Render(line)
	}
	return mutedStyle.Render(line)
}

func (m model) renderFold() string {
	a := m.analytics
	if a == nil {
		return ""
	}
	width := maxInt(40, m.width-4)
	content := m.renderAnalytics(a, width-6, true)
	content = "  " + strings.ReplaceAll(content, "\n", "\n  ")
	panel := lipgloss.NewStyle().
		Width(width).
		Padding(1, 1).
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("252")).
		Render(content)
	return "  " + strings.ReplaceAll(panel, "\n", "\n  ") + "\n"
}

func (m model) renderAnalytics(a *store.Analytics, width int, inline bool) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(a.ShortUrl))
	b.WriteString("  ")
	b.WriteString(clickStyle.Render(fmt.Sprintf("%d clicks", a.TotalClicks)))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("short URL  "))
	b.WriteString(mutedStyle.Render(truncate(m.client.BaseURL+"/r/"+a.ShortUrl, width-11)))
	if len(a.RecentClicks) == 0 {
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("No visits recorded yet."))
		return b.String()
	}
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("RECENT VISITS"))
	b.WriteString("\n")
	maxShow := 5
	if inline {
		maxShow = 3
	}
	if len(a.RecentClicks) < maxShow {
		maxShow = len(a.RecentClicks)
	}
	for _, c := range a.RecentClicks[:maxShow] {
		visit := fmt.Sprintf("%s  %-15s  %s", c.Timestamp.Format("01-02 15:04"), c.IP, c.UserAgent)
		b.WriteString(mutedStyle.Render(truncate(visit, width)))
		b.WriteString("\n")
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func (m model) renderMessages() string {
	var b strings.Builder
	if m.note != "" {
		b.WriteString("\n ")
		b.WriteString(noteStyle.Render("✓ " + m.note))
		b.WriteString("\n")
	}
	if m.err != nil {
		b.WriteString("\n ")
		b.WriteString(errStyle.Render(fmt.Sprintf("! %v", m.err)))
		b.WriteString(mutedStyle.Render("  r retry"))
		b.WriteString("\n")
	}
	return b.String()
}

func (m model) renderOverlay() string {
	var content string
	switch m.screen {
	case screenCreate:
		content = m.renderCreate()
	case screenConfirmDelete:
		content = m.renderConfirmDelete()
	case screenQr:
		content = m.renderQr()
	default:
		return ""
	}

	lines := strings.Split(content, "\n")
	contentH := len(lines)

	barH := 2
	helpH := 2
	availH := m.height - barH - helpH
	if availH < 0 {
		availH = 10
	}
	topPad := (availH - contentH) / 2
	if topPad < 0 {
		topPad = 0
	}

	var b strings.Builder
	b.WriteString(strings.Repeat("\n", topPad))
	b.WriteString(content)
	return b.String()
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

	return b.String()
}

func (m model) renderConfirmDelete() string {
	if m.cursor < 0 || m.cursor >= len(m.urls) {
		return ""
	}
	u := m.urls[m.cursor]
	var b strings.Builder
	b.WriteString(errStyle.Render(" Delete link?"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf(" %s\n", titleStyle.Render(u.ShortUrl)))
	b.WriteString(" " + mutedStyle.Render(truncate(u.LongUrl, maxInt(30, m.width-4))) + "\n\n")
	b.WriteString(" " + mutedStyle.Render("Click history will be removed. This cannot be undone.") + "\n")
	return b.String()
}

func (m model) renderQr() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" QR Code"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf(" %s\n\n", m.qrURL))
	b.WriteString(" " + strings.ReplaceAll(m.qrDisplay, "\n", "\n ") + "\n")
	return b.String()
}

func (m model) renderHelp() string {
	switch m.screen {
	case screenList:
		return helpStyle.Render(" j/k move  enter inspect  n new  c copy  v QR  d delete  r refresh  q quit  " + m.client.BaseURL)
	case screenCreate:
		return helpStyle.Render(" [tab] switch field  [enter] create URL  [esc] cancel")
	case screenConfirmDelete:
		return helpStyle.Render(" [y] confirm delete  [any other key] cancel")
	case screenQr:
		return helpStyle.Render(" [esc] back")
	}
	return ""
}

func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	if width <= 3 {
		return string(runes[:width])
	}
	return string(runes[:width-3]) + "..."
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
