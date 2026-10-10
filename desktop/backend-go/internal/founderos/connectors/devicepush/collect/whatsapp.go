package collect

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/devicepush"
)

// WhatsApp ships two macOS apps with separate group containers: the consumer
// app and WhatsApp Business (SMB). Both are looked at, and the database that
// was touched last is the app in use (lib/connectors/whatsapp.ts).
var whatsappContainers = []string{
	"group.net.whatsapp.WhatsApp.shared",    // consumer
	"group.net.whatsapp.WhatsAppSMB.shared", // Business
}

const (
	whatsappReadTimeout = 5 * time.Second
	fdaHint             = "grant Full Disk Access to the app running founderos-collector in System Settings > Privacy & Security > Full Disk Access, then restart it"
)

var coreDataEpoch = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)

// WhatsAppContainerPaths are the candidate ChatStorage.sqlite paths under home.
func WhatsAppContainerPaths(home string) []string {
	out := make([]string, 0, len(whatsappContainers))
	for _, c := range whatsappContainers {
		out = append(out, filepath.Join(home, "Library", "Group Containers", c, "ChatStorage.sqlite"))
	}
	return out
}

// ResolveChatDB picks the most recently modified existing database, or "".
// It only touches metadata, which macOS allows without Full Disk Access.
func ResolveChatDB(paths []string) string {
	type cand struct {
		path string
		mod  time.Time
	}
	var found []cand
	for _, p := range paths {
		if fi, err := os.Stat(p); err == nil {
			found = append(found, cand{p, fi.ModTime()})
		}
	}
	if len(found) == 0 {
		return ""
	}
	sort.SliceStable(found, func(i, j int) bool { return found[i].mod.After(found[j].mod) })
	return found[0].path
}

// CalibratedDate maps a ZLASTMESSAGEDATE whose unit varies across app builds:
// the largest raw value in an active database is "about now", so the scale is
// maxRaw / seconds-since-2001(now).
func CalibratedDate(raw, maxRaw float64, now time.Time) time.Time {
	elapsed := now.Sub(coreDataEpoch).Seconds()
	scale := 1.0
	if maxRaw > 0 {
		scale = maxRaw / elapsed
	}
	seconds := 0.0
	if raw > 0 {
		seconds = raw / scale
	}
	return coreDataEpoch.Add(time.Duration(seconds * float64(time.Second)))
}

type waCounts struct{ chats, unread int }

// WhatsAppStatus is the honest status of the local WhatsApp database.
func WhatsAppStatus(ctx context.Context, paths []string, now time.Time) connectors.Status {
	m := devicepush.Metas[devicepush.SourceWhatsApp]
	s := connectors.Status{ID: m.ID, Name: m.Name, Kind: m.Kind}
	dbPath := ResolveChatDB(paths)
	if dbPath == "" {
		s.State = connectors.StateNotConfigured
		s.Detail = "ChatStorage.sqlite not found. Is the WhatsApp (or WhatsApp Business) desktop app installed and signed in?"
		return s
	}
	counts, err := bounded(ctx, whatsappReadTimeout, func(ctx context.Context) (waCounts, error) {
		db, err := openReadOnly(ctx, dbPath)
		if err != nil {
			return waCounts{}, err
		}
		defer db.Close()
		var c waCounts
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM ZWACHATSESSION").Scan(&c.chats); err != nil {
			return c, err
		}
		if err := db.QueryRowContext(ctx, "SELECT COALESCE(SUM(ZUNREADCOUNT),0) FROM ZWACHATSESSION WHERE ZUNREADCOUNT>0").Scan(&c.unread); err != nil {
			return c, err
		}
		return c, nil
	})
	switch {
	case errors.Is(err, errTimedOut):
		s.State = connectors.StateError
		s.Detail = fmt.Sprintf("ChatStorage.sqlite found but the read timed out. Likely permissions: %s.", fdaHint)
	case err != nil:
		s.State = connectors.StateError
		s.Detail = "ChatStorage.sqlite exists but read failed: " + err.Error()
	default:
		mins := 0
		if fi, err := os.Stat(dbPath); err == nil {
			mins = int(math.Max(0, math.Floor(now.Sub(fi.ModTime()).Minutes()+0.5)))
		}
		source := "consumer"
		if strings.Contains(dbPath, "SMB") {
			source = "business"
		}
		s.State = connectors.StateConnected
		s.Detail = fmt.Sprintf("%d chats · %d unread · last activity %dm ago (local desktop DB, read-only)", counts.chats, counts.unread, mins)
		s.Meta = map[string]any{"chats": counts.chats, "unread": counts.unread, "source": source}
	}
	return s
}

var dialableJID = regexp.MustCompile(`^\d+@s\.whatsapp\.net$`)

const recentChatsSQL = `SELECT s.ZPARTNERNAME, s.ZCONTACTJID, s.ZLASTMESSAGEDATE, s.ZUNREADCOUNT,
  (SELECT m.ZTEXT FROM ZWAMESSAGE m WHERE m.ZCHATSESSION = s.Z_PK AND m.ZTEXT IS NOT NULL ORDER BY m.ZMESSAGEDATE DESC LIMIT 1) AS lastText,
  (SELECT MAX(m.ZMESSAGEDATE) FROM ZWAMESSAGE m WHERE m.ZCHATSESSION = s.Z_PK) AS lastDate
FROM ZWACHATSESSION s
WHERE s.ZPARTNERNAME IS NOT NULL AND (s.ZARCHIVED IS NULL OR s.ZARCHIVED = 0)
ORDER BY s.ZLASTMESSAGEDATE DESC LIMIT ?`

// RecentChats is the WhatsApp lane of the Comms feed, newest first. WhatsApp
// cannot be sent to from here; replyTo carries the number so wa.me/<digits>
// can open the exact thread (group JIDs have no dialable number and get none).
func RecentChats(ctx context.Context, dbPath string, limit int, now time.Time) ([]devicepush.Chat, error) {
	if limit <= 0 {
		limit = 40
	}
	return bounded(ctx, whatsappReadTimeout, func(ctx context.Context) ([]devicepush.Chat, error) {
		db, err := openReadOnly(ctx, dbPath)
		if err != nil {
			return nil, err
		}
		defer db.Close()
		var maxRaw float64
		if err := db.QueryRowContext(ctx, "SELECT COALESCE(MAX(ZLASTMESSAGEDATE),0) FROM ZWACHATSESSION").Scan(&maxRaw); err != nil {
			return nil, err
		}
		rows, err := db.QueryContext(ctx, recentChatsSQL, limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		out := []devicepush.Chat{}
		for rows.Next() {
			var name, jid, text sql.NullString
			var lastMsg, lastDate sql.NullFloat64
			var unread sql.NullInt64
			if err := rows.Scan(&name, &jid, &lastMsg, &unread, &text, &lastDate); err != nil {
				return nil, err
			}
			title := "Unknown chat"
			if name.Valid {
				title = name.String
			}
			c := devicepush.Chat{Source: "whatsapp", Title: title, Sender: title, Preview: truncRunes(text.String, 140), Unread: int(unread.Int64)}
			if jid.Valid && dialableJID.MatchString(jid.String) {
				c.ReplyTo = strings.SplitN(jid.String, "@", 2)[0]
			}
			if lastDate.Valid && lastDate.Float64 > 0 {
				c.TS = isoMillis(coreDataEpoch.Add(time.Duration(lastDate.Float64 * float64(time.Second))))
			} else {
				c.TS = isoMillis(CalibratedDate(lastMsg.Float64, maxRaw, now))
			}
			out = append(out, c)
		}
		return out, rows.Err()
	})
}
