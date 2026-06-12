package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shx-dow/yoorl/store"
)

type screen int

const (
	screenList screen = iota
	screenCreate
	screenConfirmDelete
)

type model struct {
	client       *Client
	screen       screen
	urls         []*store.UrlEntry
	cursor       int
	analytics    *store.Analytics
	err          error
	urlInput     textinput.Model
	aliasInput   textinput.Model
	focusedInput int
	quitting     bool
	width        int
	height       int
	note         string
	noteUntil    time.Time
}

type urlsLoadedMsg []*store.UrlEntry
type analyticsLoadedMsg *store.Analytics
type errMsg error
type tickMsg struct{}
type noteTimeoutMsg struct{}

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("33")).
			Padding(0, 2)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	detailStyle = lipgloss.NewStyle().Padding(0, 1)

	errStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("33"))

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("33"))

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("33")).
			Bold(true)

	hashStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("220"))

	noteStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Bold(true)
)

func initialModel(client *Client) model {
	urlIn := textinput.New()
	urlIn.Placeholder = "https://example.com"
	urlIn.Width = 60
	urlIn.Focus()

	aliasIn := textinput.New()
	aliasIn.Placeholder = "my-custom-alias (optional)"
	aliasIn.Width = 60

	return model{
		client:       client,
		screen:       screenList,
		urlInput:     urlIn,
		aliasInput:   aliasIn,
		focusedInput: 0,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		loadURLs(m.client),
		tick(),
	)
}

func loadURLs(client *Client) tea.Cmd {
	return func() tea.Msg {
		urls, err := client.ListURLs()
		if err != nil {
			return errMsg(err)
		}
		return urlsLoadedMsg(urls)
	}
}

func loadAnalytics(client *Client, shortURL string) tea.Cmd {
	return func() tea.Msg {
		a, err := client.GetAnalytics(shortURL)
		if err != nil {
			return errMsg(err)
		}
		return analyticsLoadedMsg(a)
	}
}

func tick() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m *model) setNote(msg string, duration time.Duration) {
	m.note = msg
	m.noteUntil = time.Now().Add(duration)
}
