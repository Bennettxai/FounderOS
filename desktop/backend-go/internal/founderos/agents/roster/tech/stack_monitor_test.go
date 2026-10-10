package tech

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/devicepush"
)

func push(t *testing.T, r *devicepush.Receiver, device, label string, statuses ...connectors.Status) {
	t.Helper()
	if err := r.Accept(context.Background(), devicepush.Payload{Device: device, Label: label, CapturedAt: time.Now().UTC().Format(time.RFC3339), Statuses: statuses}); err != nil {
		t.Fatal(err)
	}
}

func stackStatus(state connectors.State, detail string) connectors.Status {
	return connectors.Status{ID: "local-stack", Name: "Local Stack", Kind: connectors.KindLocal, State: state, Detail: detail,
		Meta: map[string]any{"paperclip": "up · agent board", "ffmpeg": "down"}}
}

func wisprStatus(state connectors.State, detail string) connectors.Status {
	return connectors.Status{ID: "wispr", Name: "Wispr Flow", Kind: connectors.KindLocal, State: state, Detail: detail,
		Meta: map[string]any{"dictations": 12}}
}

func TestStackMonitorMeta(t *testing.T) {
	m := (&StackMonitor{}).Meta()
	if m.ID != "stack-monitor" || m.Name != "Stack Monitor" || m.DepartmentID != "dept-tech" {
		t.Fatalf("meta = %+v", m)
	}
}

func TestStackMonitorNoReceiver(t *testing.T) {
	res, _ := (&StackMonitor{}).Run(context.Background())
	if res.OK || !strings.Contains(res.Summary, "device push receiver is not configured") {
		t.Fatalf("res = %+v", res)
	}
}

func TestStackMonitorNoPushYet(t *testing.T) {
	r := devicepush.NewReceiver(devicepush.NewMemStore(), "alexs-macbook-pro")
	r.Host = "mini"
	res, _ := (&StackMonitor{Devices: r}).Run(context.Background())
	if res.OK {
		t.Fatalf("no push is not a healthy stack: %s", res.Summary)
	}
	for _, want := range []string{"the host mini has not pushed Local Stack", "Wispr: not_configured", "alexs-macbook-pro not_configured"} {
		if !strings.Contains(res.Summary, want) {
			t.Errorf("summary missing %q: %s", want, res.Summary)
		}
	}
}

func TestStackMonitorFreshPush(t *testing.T) {
	r := devicepush.NewReceiver(devicepush.NewMemStore(), "alexs-macbook-pro", "mini")
	r.Host = "mini"
	push(t, r, "mini", "Mac mini",
		stackStatus(connectors.StateConnected, "6/7 up — paperclip, hermes · down: ffmpeg"),
		wisprStatus(connectors.StateConnected, "12 dictations"))
	res, _ := (&StackMonitor{Devices: r}).Run(context.Background())
	if !res.OK {
		t.Fatalf("fresh connected stack: %s", res.Summary)
	}
	for _, want := range []string{"6/7 up — paperclip, hermes · down: ffmpeg · pushed by Mac mini", "Wispr: 12 dictations", "mini connected", "alexs-macbook-pro not_configured"} {
		if !strings.Contains(res.Summary, want) {
			t.Errorf("summary missing %q: %s", want, res.Summary)
		}
	}
	data, _ := res.Data.(map[string]any)
	if data == nil || data["stack"] == nil || data["wispr"] == nil || data["devices"] == nil {
		t.Errorf("data = %+v", res.Data)
	}
}

func TestStackMonitorStackDown(t *testing.T) {
	r := devicepush.NewReceiver(devicepush.NewMemStore())
	r.Host = "mini"
	push(t, r, "mini", "Mac mini", stackStatus(connectors.StateError, "0/7 up — · down: paperclip, hermes"))
	res, _ := (&StackMonitor{Devices: r}).Run(context.Background())
	if res.OK || !strings.Contains(res.Summary, "0/7 up") {
		t.Fatalf("stack down: %s", res.Summary)
	}
}

func TestStackMonitorStalePush(t *testing.T) {
	r := devicepush.NewReceiver(devicepush.NewMemStore())
	r.Host = "mini"
	r.StaleAfter = -time.Second // every push is already stale
	push(t, r, "mini", "Mac mini", stackStatus(connectors.StateConnected, "7/7 up"))
	res, _ := (&StackMonitor{Devices: r}).Run(context.Background())
	if res.OK || !strings.Contains(res.Summary, "stale") {
		t.Fatalf("a stale push must not read as a live stack: %s", res.Summary)
	}
}

// FounderOS v1 checked the machine it ran on. A healthy MacBook push must not
// cover for the host (the mini) whose services are down.
func TestStackMonitorJudgesTheHostNotTheBestMac(t *testing.T) {
	r := devicepush.NewReceiver(devicepush.NewMemStore(), "alexs-macbook-pro", "mini")
	r.Host = "mini"
	push(t, r, "alexs-macbook-pro", "MacBook", stackStatus(connectors.StateConnected, "7/7 up"))
	push(t, r, "mini", "Mac mini", stackStatus(connectors.StateError, "2/7 up — · down: paperclip, hermes"))
	res, _ := (&StackMonitor{Devices: r}).Run(context.Background())
	if res.OK || !strings.Contains(res.Summary, "2/7 up") {
		t.Fatalf("the host's broken stack must fail the check: %s", res.Summary)
	}
}
