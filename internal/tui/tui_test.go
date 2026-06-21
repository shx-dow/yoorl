package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/shx-dow/yoorl/internal/client"
	"github.com/shx-dow/yoorl/store"
	"github.com/stretchr/testify/require"
)

func TestAnalyticsFoldIsClosedByDefaultAtWideWidth(t *testing.T) {
	m := dashboardTestModel(140)

	output := m.renderList()

	require.NotContains(t, output, "RECENT VISITS")
	require.NotContains(t, output, "SELECTED LINK")
}

func TestEnterOpensAnalyticsInlineAtWideWidth(t *testing.T) {
	m := dashboardTestModel(140)

	updated, _ := m.handleListKey(tea.KeyMsg{Type: tea.KeyEnter})
	result := updated.(model)

	require.True(t, result.foldOpen)
	require.Contains(t, result.renderList(), "RECENT VISITS")
	require.Contains(t, result.renderList(), "launch-26")
}

func TestMovingSelectionClosesAnalyticsFold(t *testing.T) {
	m := dashboardTestModel(100)
	m.foldOpen = true

	updated, _ := m.handleListKey(tea.KeyMsg{Type: tea.KeyDown})
	result := updated.(model)

	require.False(t, result.foldOpen)
	require.Equal(t, 1, result.cursor)
}

func dashboardTestModel(width int) model {
	return model{
		client: &Client{Client: client.New("http://localhost:8080", "")},
		width:  width,
		urls: []*store.UrlEntry{
			{ShortUrl: "launch-26", LongUrl: "https://yoorl.dev/releases/summer", TotalClicks: 842},
			{ShortUrl: "docs", LongUrl: "https://github.com/shx-dow/yoorl/wiki", TotalClicks: 203},
		},
		analytics: &store.Analytics{
			ShortUrl:    "launch-26",
			TotalClicks: 842,
			RecentClicks: []store.ClickEvent{
				{IP: "10.24.8.19", UserAgent: "Firefox / Linux"},
			},
		},
	}
}
