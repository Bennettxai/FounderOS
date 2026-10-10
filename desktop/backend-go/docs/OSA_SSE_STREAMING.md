# OSA SSE Streaming Implementation

## Overview

Real-time Server-Sent Events (SSE) streaming system for OSA build progress updates. Provides live build status, progress percentages, and error messages to frontend clients.

## Architecture

```
OSA-5 Webhook → OSAWebhooksHandler → BuildEventBus → OSAStreamingHandler → SSE Clients
                      ↓
                  Database
```

### Components

1. **BuildEventBus** (`internal/services/build_event_bus.go`)
   - Pub/sub event distribution system
   - Thread-safe subscriber management
   - Automatic cleanup on disconnect
   - Buffered channels to prevent blocking

2. **OSAStreamingHandler** (`internal/handlers/osa_streaming.go`)
   - SSE endpoint handler
   - Connection management
   - Heartbeat/keep-alive mechanism
   - Graceful disconnection handling

3. **OSAWebhooksHandler** (updated in `internal/handlers/osa_webhooks.go`)
   - Receives webhooks from OSA-5
   - Stores events in database
   - Broadcasts to event bus

## API Endpoints

### Stream Build Progress
```http
GET /api/osa/stream/build/:app_id
Authorization: Bearer <token>
```

**Response (SSE Stream):**
```
data: {"type":"connected","app_id":"..."}

data: {"id":"...","app_id":"...","event_type":"build_started","phase":"building","progress_percent":0,"status_message":"Starting build...","timestamp":"2026-01-11T..."}

data: {"id":"...","app_id":"...","event_type":"build_progress","phase":"building","progress_percent":25,"status_message":"Installing dependencies...","timestamp":"2026-01-11T..."}

data: {"id":"...","app_id":"...","event_type":"build_progress","phase":"building","progress_percent":75,"status_message":"Compiling code...","timestamp":"2026-01-11T..."}

data: {"id":"...","app_id":"...","event_type":"build_complete","phase":"completion","progress_percent":100,"status_message":"Build completed successfully","timestamp":"2026-01-11T..."}
```

**Event Format:**
```typescript
interface BuildEvent {
  id: string;                    // Event ID
  app_id: string;               // App being built
  workspace_id?: string;        // Optional workspace ID
  event_type: string;           // build_started, build_progress, build_complete, build_failed
  phase?: string;               // building, completion
  progress_percent: number;     // 0-100
  status_message?: string;      // Human-readable status
  data?: Record<string, any>;   // Additional metadata
  timestamp: string;            // ISO 8601 timestamp
}
```

### Get Stream Statistics
```http
GET /api/osa/stream/stats
Authorization: Bearer <token>
```

**Response:**
```json
{
  "total_subscribers": 5,
  "timestamp": "2026-01-11T...",
  "user_id": "..."
}
```

### Get App Stream Statistics
```http
GET /api/osa/stream/stats/:app_id
Authorization: Bearer <token>
```

**Response:**
```json
{
  "app_id": "...",
  "subscriber_count": 2,
  "timestamp": "2026-01-11T...",
  "user_id": "..."
}
```

## Usage Examples

### Frontend (JavaScript)

```javascript
// Connect to SSE stream
const appId = 'your-app-id-here';
const eventSource = new EventSource(
  `/api/osa/stream/build/${appId}`,
  {
    headers: {
      'Authorization': `Bearer ${token}`
    }
  }
);

// Handle build events
eventSource.onmessage = (event) => {
  const buildEvent = JSON.parse(event.data);

  if (buildEvent.type === 'connected') {
    console.log('Connected to build stream');
    return;
  }

  console.log(`Build progress: ${buildEvent.progress_percent}%`);
  console.log(`Status: ${buildEvent.status_message}`);

  // Update UI
  updateProgressBar(buildEvent.progress_percent);
  updateStatusMessage(buildEvent.status_message);

  // Handle completion
  if (buildEvent.event_type === 'build_complete') {
    console.log('Build completed!');
    eventSource.close();
  }

  // Handle errors
  if (buildEvent.event_type === 'build_failed') {
    console.error('Build failed:', buildEvent.status_message);
    eventSource.close();
  }
};

eventSource.onerror = (error) => {
  console.error('SSE connection error:', error);
  eventSource.close();
};

// Clean up on page unload
window.addEventListener('beforeunload', () => {
  eventSource.close();
});
```

### Frontend (TypeScript + React)

```typescript
import { useEffect, useState } from 'react';

interface BuildEvent {
  id: string;
  app_id: string;
  event_type: string;
  phase?: string;
  progress_percent: number;
  status_message?: string;
  timestamp: string;
}

export function useBuildProgress(appId: string, token: string) {
  const [progress, setProgress] = useState(0);
  const [status, setStatus] = useState('');
  const [isComplete, setIsComplete] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const eventSource = new EventSource(
      `/api/osa/stream/build/${appId}`,
      {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      }
    );

    eventSource.onmessage = (event) => {
      const buildEvent: BuildEvent = JSON.parse(event.data);

      if (buildEvent.type === 'connected') return;

      setProgress(buildEvent.progress_percent);
      setStatus(buildEvent.status_message || '');

      if (buildEvent.event_type === 'build_complete') {
        setIsComplete(true);
        eventSource.close();
      }

      if (buildEvent.event_type === 'build_failed') {
        setError(buildEvent.status_message || 'Build failed');
        eventSource.close();
      }
    };

    eventSource.onerror = () => {
      setError('Connection lost');
      eventSource.close();
    };

    return () => {
      eventSource.close();
    };
  }, [appId, token]);

  return { progress, status, isComplete, error };
}
```

## OSA-5 Webhook Integration

### Webhook Endpoint

OSA-5 sends build events to:
```
POST /api/osa/webhooks/build-event
X-OSA-Signature: <hmac-sha256-signature>
```

**Payload:**
```json
{
  "event_type": "build_progress",
  "workflow_id": "workflow-123",
  "timestamp": "2026-01-11T...",
  "status": "in_progress",
  "data": {
    "progress": 50,
    "message": "Compiling TypeScript...",
    "phase": "building"
  }
}
```

### Event Flow

1. OSA-5 sends webhook to `/api/osa/webhooks/build-event`
2. `OSAWebhooksHandler.HandleBuildEvent()` validates HMAC signature
3. Event stored in `osa_build_events` table
4. Event broadcast to all subscribers via `BuildEventBus`
5. Connected SSE clients receive event in real-time

## Configuration

### Environment Variables

```bash
# OSA webhook secret for HMAC verification
OSA_SHARED_SECRET=your-webhook-secret-here

# Enable OSA integration
OSA_ENABLED=true
```

### HMAC Signature Verification

OSA-5 signs webhooks with HMAC-SHA256:

```go
mac := hmac.New(sha256.New, []byte(webhookSecret))
mac.Write(body)
expectedMAC := hex.EncodeToString(mac.Sum(nil))
```

In development, unsigned webhooks are allowed. In production, set a strong `OSA_SHARED_SECRET`.

## Database Schema

Build events are stored in `osa_build_events`:

```sql
CREATE TABLE osa_build_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id UUID NOT NULL REFERENCES osa_generated_apps(id) ON DELETE CASCADE,
    workspace_id UUID REFERENCES osa_workspaces(id) ON DELETE SET NULL,
    event_type VARCHAR(50) NOT NULL,
    event_data JSONB DEFAULT '{}',
    build_id VARCHAR(255),
    phase VARCHAR(50),
    progress_percent INTEGER DEFAULT 0,
    status_message TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

## Performance Characteristics

- **Buffered Channels**: 100-event buffer per subscriber prevents blocking
- **Heartbeat**: 30-second keep-alive messages prevent connection timeout
- **Automatic Cleanup**: Subscribers removed on context cancellation
- **Thread-Safe**: Concurrent-safe subscriber management with RWMutex
- **Memory Efficient**: No history retention, events only in database

## Error Handling

### Client Disconnection

Automatically detected via:
- Context cancellation
- Write errors
- Channel closure

### Full Buffers

If subscriber channel is full (100 events), new events are dropped with warning log.

### Invalid App ID

Returns 400 Bad Request if app ID is malformed.

### Unauthorized

Returns 401 if user not authenticated.

## Monitoring

### Subscriber Statistics

```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/osa/stream/stats

# Response:
{
  "total_subscribers": 3,
  "timestamp": "2026-01-11T..."
}
```

### App-Specific Statistics

```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/osa/stream/stats/app-id-here

# Response:
{
  "app_id": "...",
  "subscriber_count": 1,
  "timestamp": "2026-01-11T..."
}
```

## Testing

### Manual Testing

```bash
# Terminal 1: Start server
cd desktop/backend-go
go run ./cmd/server

# Terminal 2: Subscribe to stream
curl -N -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/osa/stream/build/your-app-id

# Terminal 3: Trigger webhook
curl -X POST http://localhost:8080/api/osa/webhooks/build-event \
  -H "Content-Type: application/json" \
  -d '{
    "event_type": "build_progress",
    "workflow_id": "workflow-123",
    "timestamp": "2026-01-11T12:00:00Z",
    "status": "in_progress",
    "data": {
      "progress": 50,
      "message": "Building..."
    }
  }'
```

### Unit Test Example

```go
func TestBuildEventBus(t *testing.T) {
    logger := slog.Default()
    bus := services.NewBuildEventBus(logger)

    ctx := context.Background()
    appID := uuid.New()
    userID := uuid.New()

    // Subscribe
    sub := bus.Subscribe(ctx, userID, appID)

    // Publish event
    event := services.BuildEvent{
        ID:              uuid.New(),
        AppID:           appID,
        EventType:       "build_progress",
        ProgressPercent: 50,
        StatusMessage:   "Building...",
    }
    bus.Publish(event)

    // Receive event
    select {
    case received := <-sub.Events:
        assert.Equal(t, event.ProgressPercent, received.ProgressPercent)
    case <-time.After(1 * time.Second):
        t.Fatal("timeout waiting for event")
    }

    // Cleanup
    bus.Unsubscribe(sub.ID)
}
```

## Troubleshooting

### No events received

1. Check authentication token is valid
2. Verify app ID exists in database
3. Check webhook signature is valid
4. Ensure `OSA_ENABLED=true`

### Connection drops

1. Check network stability
2. Verify heartbeat is being sent (every 30s)
3. Check for nginx/proxy buffering (`X-Accel-Buffering: no`)

### Events delayed

1. Check subscriber channel buffer (100 events max)
2. Verify database write performance
3. Check event bus publish performance

## Future Enhancements

1. **Event Filtering**: Allow clients to filter by event type
2. **History Replay**: Send recent events on connect
3. **Compression**: Gzip SSE stream for bandwidth efficiency
4. **Redis Pub/Sub**: Scale across multiple backend instances
5. **Metrics**: Prometheus metrics for subscriber count, event rate

## Related Files

- `internal/services/build_event_bus.go` - Event bus implementation
- `internal/handlers/osa_streaming.go` - SSE endpoint handler
- `internal/handlers/osa_webhooks.go` - Webhook receiver (updated)
- `internal/handlers/handlers.go` - Route registration (updated)
- `cmd/server/main.go` - Service initialization (updated)

## References

- [MDN: Server-Sent Events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events)
- [EventSource API](https://developer.mozilla.org/en-US/docs/Web/API/EventSource)
- [OSA-5 Integration Guide](./OSA_INTEGRATION_COMPLETE_STATUS.md)
