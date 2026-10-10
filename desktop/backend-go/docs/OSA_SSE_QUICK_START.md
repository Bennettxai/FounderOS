# OSA SSE Streaming - Quick Start Guide

## TL;DR

Real-time build progress updates via Server-Sent Events (SSE).

## Frontend Integration (5 minutes)

### JavaScript
```javascript
// 1. Connect to stream
const eventSource = new EventSource(
  `/api/osa/stream/build/${appId}`,
  { headers: { 'Authorization': `Bearer ${token}` } }
);

// 2. Handle events
eventSource.onmessage = (event) => {
  const { progress_percent, status_message, event_type } = JSON.parse(event.data);

  // Update UI
  updateProgress(progress_percent);
  updateStatus(status_message);

  // Handle completion
  if (event_type === 'build_complete') {
    eventSource.close();
  }
};

// 3. Handle errors
eventSource.onerror = () => {
  console.error('Connection lost');
  eventSource.close();
};

// 4. Cleanup
window.addEventListener('beforeunload', () => eventSource.close());
```

### React Hook
```typescript
function useBuildProgress(appId: string) {
  const [progress, setProgress] = useState(0);
  const [status, setStatus] = useState('');
  const [done, setDone] = useState(false);

  useEffect(() => {
    const es = new EventSource(`/api/osa/stream/build/${appId}`);

    es.onmessage = (e) => {
      const event = JSON.parse(e.data);
      setProgress(event.progress_percent);
      setStatus(event.status_message);
      if (event.event_type === 'build_complete') {
        setDone(true);
        es.close();
      }
    };

    return () => es.close();
  }, [appId]);

  return { progress, status, done };
}

// Usage
function BuildProgress({ appId }) {
  const { progress, status } = useBuildProgress(appId);
  return (
    <div>
      <progress value={progress} max={100} />
      <p>{status}</p>
    </div>
  );
}
```

## API Endpoints

### Stream Build Progress
```
GET /api/osa/stream/build/:app_id
Authorization: Bearer <token>
```

**Response (SSE):**
```
data: {"type":"connected","app_id":"..."}

data: {"progress_percent":25,"status_message":"Installing dependencies..."}

data: {"progress_percent":75,"status_message":"Compiling code..."}

data: {"progress_percent":100,"status_message":"Build complete"}
```

### Get Statistics
```bash
# Global stats
curl -H "Authorization: Bearer $TOKEN" \
  /api/osa/stream/stats

# Per-app stats
curl -H "Authorization: Bearer $TOKEN" \
  /api/osa/stream/stats/$APP_ID
```

## Event Types

| Event Type | Description | Progress |
|------------|-------------|----------|
| `build_started` | Build initiated | 0% |
| `build_progress` | Build in progress | 1-99% |
| `build_complete` | Build succeeded | 100% |
| `build_failed` | Build failed | N/A |

## Event Schema

```typescript
interface BuildEvent {
  id: string;                    // Event UUID
  app_id: string;               // App UUID
  workspace_id?: string;        // Workspace UUID
  event_type: string;           // See Event Types above
  phase?: string;               // "building" or "completion"
  progress_percent: number;     // 0-100
  status_message?: string;      // Human-readable status
  data?: Record<string, any>;   // Additional metadata
  timestamp: string;            // ISO 8601
}
```

## OSA-5 Webhook Configuration

```bash
# Set webhook endpoint
WEBHOOK_URL=https://your-backend.com/api/osa/webhooks/build-event

# Set shared secret (for HMAC verification)
WEBHOOK_SECRET=your-secret-here
```

## Testing

### 1. Start Server
```bash
cd desktop/backend-go
go run ./cmd/server
```

### 2. Subscribe to Stream
```bash
curl -N -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/osa/stream/build/$APP_ID
```

### 3. Trigger Webhook (Simulate OSA-5)
```bash
curl -X POST http://localhost:8080/api/osa/webhooks/build-event \
  -H "Content-Type: application/json" \
  -d '{
    "event_type": "build_progress",
    "workflow_id": "workflow-123",
    "timestamp": "2026-01-11T12:00:00Z",
    "status": "in_progress",
    "data": {
      "progress": 50,
      "message": "Compiling TypeScript..."
    }
  }'
```

## Troubleshooting

### No Events Received
- ✅ Check JWT token is valid
- ✅ Verify app ID exists
- ✅ Check webhook signature

### Connection Drops
- ✅ Heartbeat sent every 30s
- ✅ Check nginx buffering (`X-Accel-Buffering: no`)
- ✅ Verify network stability

### Events Delayed
- ✅ Check subscriber buffer (100 events max)
- ✅ Verify database performance
- ✅ Monitor event bus latency

## Key Files

- `internal/services/build_event_bus.go` - Event bus
- `internal/handlers/osa_streaming.go` - SSE handler
- `docs/OSA_SSE_STREAMING.md` - Full documentation

## Environment Variables

```bash
# Enable OSA integration
OSA_ENABLED=true

# Webhook HMAC secret
OSA_SHARED_SECRET=your-webhook-secret-here
```

## Performance Notes

- **Buffer Size**: 100 events per subscriber
- **Heartbeat**: 30 seconds
- **Auto-cleanup**: On disconnect
- **Thread-safe**: Concurrent subscribers supported

## Complete Example

```typescript
import { useEffect, useState } from 'react';

export function BuildMonitor({ appId }: { appId: string }) {
  const [progress, setProgress] = useState(0);
  const [status, setStatus] = useState('Initializing...');
  const [isComplete, setIsComplete] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const token = localStorage.getItem('token');
    const eventSource = new EventSource(
      `/api/osa/stream/build/${appId}`,
      { headers: { 'Authorization': `Bearer ${token}` } }
    );

    eventSource.onmessage = (event) => {
      const buildEvent = JSON.parse(event.data);

      // Skip connection confirmation
      if (buildEvent.type === 'connected') return;

      // Update state
      setProgress(buildEvent.progress_percent);
      setStatus(buildEvent.status_message || '');

      // Handle completion
      if (buildEvent.event_type === 'build_complete') {
        setIsComplete(true);
        eventSource.close();
      }

      // Handle failure
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
  }, [appId]);

  if (error) {
    return <div className="error">Error: {error}</div>;
  }

  if (isComplete) {
    return <div className="success">Build Complete! 🎉</div>;
  }

  return (
    <div className="build-monitor">
      <progress value={progress} max={100} />
      <p>{progress}% - {status}</p>
    </div>
  );
}
```

## Next Steps

1. Integrate into your frontend
2. Configure OSA-5 webhooks
3. Test end-to-end flow
4. Monitor subscriber statistics

## Support

See full documentation: `docs/OSA_SSE_STREAMING.md`
