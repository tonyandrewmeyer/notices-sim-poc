package main

import (
	"time"
	"context"
	"strconv"
	"log/slog"
	"os/exec"

	"github.com/canonical/pebble/client"
)

type PebbleClient interface {
	CloseIdleConnections()
	WaitNotices(ctx context.Context, serverTimeout time.Duration, opts *client.NoticesOptions) ([]*client.Notice, error)
}

type WorkloadEventType int

const (
	CustomNoticeEvent WorkloadEventType = iota
	ChangeUpdatedEvent
	RecoverCheckEvent
	PerformCheckEvent
)

type WorkloadEvent struct {
	Type         WorkloadEventType
	NoticeID     string
	NoticeType   string
	NoticeKey    string
}

func (e WorkloadEventType) String() string {
	switch e {
	case CustomNoticeEvent:
		return "custom"
	case ChangeUpdatedEvent:
		return "change-updated"
	case RecoverCheckEvent:
		return "recover-check"
	case PerformCheckEvent:
		return "perform-check"
	}
	return "unknown"
}

type WorkloadEvents interface {
	AddWorkloadEvent(evt WorkloadEvent) string
	RemoveWorkloadEvent(id string)
	HasWorkloadEvent(noticeType client.NoticeType, noticeKey string) bool
}

type workloadEvents struct {
	nextID  int
	pending map[string]WorkloadEvent
}

func NewWorkloadEvents() WorkloadEvents {
	return &workloadEvents{pending: make(map[string]WorkloadEvent)}
}

func (c *workloadEvents) AddWorkloadEvent(evt WorkloadEvent) string {
	id := strconv.Itoa(c.nextID)
	c.nextID++
	c.pending[id] = evt
	return id
}

func (c *workloadEvents) RemoveWorkloadEvent(id string) {
	delete(c.pending, id)
}

func (c *workloadEvents) HasWorkloadEvent(noticeType client.NoticeType, noticeKey string) bool {
	// tam: Currently very inefficient. If we actually want to do this then we can
	// easily have a O(1) lookup with a small amount of extra memory.
	for _, v := range c.pending {
		if v.NoticeType == string(noticeType) && v.NoticeKey == noticeKey {
			return true
		}
	}
	return false
}

type pebbleNoticer struct {
	workloadEvents  WorkloadEvents
	pebbleClient PebbleClient
}

func main() {
	config := client.Config{
		Socket: "/tmp/pebble/.pebble.socket",
	}
	pebbleClient, err := client.New(&config)
	if err != nil {
		slog.Error("failed to create pebble client", "err", err)
		return
	}
	defer pebbleClient.CloseIdleConnections()
	workloadEvents := NewWorkloadEvents()
	noticer := &pebbleNoticer{
		workloadEvents:  workloadEvents,
		pebbleClient: pebbleClient,
	}

	// Allow 1000 events to be buffered.
	charm := make (chan WorkloadEvent, 1000)
	// Emit events to the charm as they come in.
	go emitter(charm)

	// Loop forever waiting for new notices.
	noticer.run(charm)
}

func (n *pebbleNoticer) run(charm chan WorkloadEvent) () {
	const (
		waitTimeout = 30 * time.Second
		errorDelay  = time.Second
	)

	slog.Info("pebbleNoticer starting")
	defer slog.Info("pebbleNoticer stopped")

	var after time.Time
	ctx := context.Background()
	for {
		options := &client.NoticesOptions{After: after}
		notices, err := n.pebbleClient.WaitNotices(ctx, waitTimeout, options)
		if err != nil {
			slog.Error("failed to get notices", "err", err)
			return
		}

		for _, notice := range notices {
			err := n.processNotice(charm, notice)
			if err != nil {
				slog.Error("failed to process notice", "err", err)
				return
			}
			after = notice.LastRepeated
		}
	}
}

func (n *pebbleNoticer) processNotice(charm chan WorkloadEvent, notice *client.Notice) error {
	var eventType WorkloadEventType
	switch notice.Type {
	case client.CustomNotice:
		eventType = CustomNoticeEvent
	// tam: new notice type for change-update
	case client.ChangeUpdateNotice:
		// tam: special-case recover-check and perform-check
		switch {
		case notice.LastData["kind"] == "recover-check":
			eventType = RecoverCheckEvent
		case notice.LastData["kind"] == "perform-check":
			eventType = PerformCheckEvent
		default:
			eventType = ChangeUpdatedEvent
		}
	default:
		slog.Info("ignoring notice", "type", notice.Type, "key", notice.Key)
		return nil
	}

	// tam: If there is already an item with the same notice type and key in
	// n.workloadEvents, we want to log this and return
	// Disabled because we aren't getting duplicates: why?
	//if n.workloadEvents.HasWorkloadEvent(notice.Type, notice.Key) {
	//	slog.Info("ignoring duplicate notice", "type", notice.Type, "key", notice.Key)
	//	return nil
	//}

	event := WorkloadEvent{
		Type:         eventType,
		NoticeID:     notice.ID,
		NoticeType:   string(notice.Type),
		NoticeKey:    notice.Key,
	}
	eventID := n.workloadEvents.AddWorkloadEvent(event)
	defer n.workloadEvents.RemoveWorkloadEvent(eventID)

	// tam: Send the event to the charm!
	slog.Info("sending Juju event", "type", event.Type, "notice-type", event.NoticeType, "key", event.NoticeKey, "id", event.NoticeID)
	charm <- event
	return nil
}

func emitter(charm chan WorkloadEvent) {
	for {
		event := <-charm
		slog.Info("processing event in charm", "event-type", event.Type, "notice-type", event.NoticeType, "notice-key", event.NoticeKey, "notice-id", event.NoticeID)
		out, err := exec.Command(".venv/bin/python", "./charm.py", event.Type.String(), event.NoticeID, event.NoticeType, event.NoticeKey).CombinedOutput()
		if err != nil {
			slog.Error("could not execute charm", "err", err, "output", out)
		}
	}
}
