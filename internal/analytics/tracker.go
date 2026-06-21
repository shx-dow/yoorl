package analytics

import (
	"sync/atomic"

	"github.com/rs/zerolog"
	"github.com/shx-dow/yoorl/store"
)

type clickTask struct {
	ShortUrl string
	Event    store.ClickEvent
}

type Tracker struct {
	events      chan clickTask
	done        chan struct{}
	dropped     atomic.Int64
	processed   atomic.Int64
}

func NewTracker() *Tracker {
	return &Tracker{
		events: make(chan clickTask, 1024),
		done:   make(chan struct{}),
	}
}

func (t *Tracker) Start(log zerolog.Logger) {
	go func() {
		for task := range t.events {
			if err := store.RecordClick(task.ShortUrl, task.Event); err != nil {
				log.Error().Err(err).Str("short_url", task.ShortUrl).Msg("failed to record click")
			}
			t.processed.Add(1)
		}
		close(t.done)
	}()
	log.Info().Int("buffer", cap(t.events)).Msg("analytics tracker started")
}

func (t *Tracker) Stop() {
	close(t.events)
	<-t.done
}

func (t *Tracker) Track(shortUrl string, event store.ClickEvent) {
	select {
	case t.events <- clickTask{ShortUrl: shortUrl, Event: event}:
	default:
		t.dropped.Add(1)
	}
}

func (t *Tracker) Stats() (dropped, processed int64) {
	return t.dropped.Load(), t.processed.Load()
}
