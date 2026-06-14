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
	screenQr
)

type model struct {
	client       *Client
	screen       screen
	urls         []*store.UrlEntry
	cursor       int
	foldOpen     bool
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
	qrDisplay    string
	qrURL        string
}

type urlsLoadedMsg []*store.UrlEntry
type analyticsLoadedMsg *store.Analytics
type errMsg error
type tickMsg struct{}
type noteTimeoutMsg struct{}
type healthCheckedMsg struct{}

var (
	statsBarStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("223"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	errStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("203")).
			Bold(true)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("215"))

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("223")).
			Background(lipgloss.Color("237"))

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("215")).
			Bold(true)

	hashStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("215"))

	noteStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("150")).
			Bold(true)

	mutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	clickStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("222"))

	borderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("239"))
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
		checkHealth(m.client),
		loadURLs(m.client),
		tick(),
	)
}

func checkHealth(client *Client) tea.Cmd {
	return func() tea.Msg {
		if err := client.Health(); err != nil {
			return errMsg(err)
		}
		return healthCheckedMsg{}
	}
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
