package tech

import (
	"context"
	"fmt"
	"strings"

	"github.com/rhl/businessos-backend/internal/founderos/agents"
	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/devicepush"
)

// StackMonitor reports the local stack and Wispr Flow as the Macs pushed
// them (founderos-collector), plus how fresh each Mac's push is. The backend
// never probes a device port or path itself.
type StackMonitor struct {
	Devices *devicepush.Receiver
}

func (a *StackMonitor) Meta() agents.Meta {
	return agents.Meta{
		ID:           "stack-monitor",
		Name:         "Stack Monitor",
		Description:  "Live check of the local creative/infra stack: Remotion, Ollama, command-center, OpenClaw, tmux, whisper, ffmpeg, higgsfield, gh.",
		DepartmentID: "dept-tech",
	}
}

func (a *StackMonitor) Run(ctx context.Context) (agents.Result, error) {
	if a.Devices == nil {
		return agents.Result{OK: false, Summary: "device push receiver is not configured on the bridge; no Mac can report its stack"}, nil
	}
	stack := a.Devices.HostConnector(devicepush.SourceLocalStack).Status(ctx) // the host, as FounderOS v1 checked its own machine
	wispr := a.Devices.Connector(devicepush.SourceWispr).Status(ctx)

	wisprPart := string(wispr.State)
	if wispr.State == connectors.StateConnected {
		wisprPart = wispr.Detail
	}
	summary := fmt.Sprintf("%s · Wispr: %s", stack.Detail, wisprPart)

	devices, err := a.Devices.Devices(ctx)
	if err != nil {
		summary += " · devices: unreadable (" + err.Error() + ")"
	} else if len(devices) > 0 {
		parts := make([]string, len(devices))
		for i, d := range devices {
			parts[i] = fmt.Sprintf("%s %s", d.Device, d.State)
		}
		summary += " · devices: " + strings.Join(parts, ", ")
	}
	return agents.Result{
		OK:      stack.State == connectors.StateConnected,
		Summary: summary,
		Data:    map[string]any{"stack": stack.Meta, "wispr": wispr.Meta, "devices": devices, "stackState": stack.State, "wisprState": wispr.State},
	}, nil
}
