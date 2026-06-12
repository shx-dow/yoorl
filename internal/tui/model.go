package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/table"
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
	client    *Client
	screen    screen
	urls      []*store.UrlEntry
	table     table.Model
	analytics *store.Analytics
	err       error
	textInput textinput.Model
	quitting  bool
	width     int
	height    int
}

type urlsLoadedMsg []*store.UrlEntry
type analyticsLoadedMsg *store.Analytics
type errMsg error
type tickMsg struct{}

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("33")).
			Padding(0, 2)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	detailStyle = lipgloss.NewStyle().
			Padding(0, 1)

	errStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("33"))
)

func initialModel(client *Client) model {
	t := table.New(
		table.WithColumns([]table.Column{
			{Title: "#", Width: 4},
			{Title: "Short URL", Width: 12},
			{Title: "Destination", Width: 50},
			{Title: "Clicks", Width: 8},
		}),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("33")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("33")).
		Bold(false)
	t.SetStyles(s)

	ti := textinput.New()
	ti.Placeholder = "https://example.com"
	ti.Width = 60

	return model{
		client:    client,
		screen:    screenList,
		table:     t,
		textInput: ti,
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
