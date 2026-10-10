package api

import (
	"bytes"
	"context"

	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/beehiiv"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/manychat"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/zernio"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
	"github.com/rhl/businessos-backend/internal/founderos/pages/pagekit"
	"github.com/rhl/businessos-backend/internal/founderos/pages/social"
)

// /os/social, /os/social/[platform] and /os/social/beehiiv (spec 6.3): the
// FounderOS v1 /api/social/* routes at their route-map bridge paths, plus the
// page view models. Reads are Postgres (personal) and the Zernio,
// Beehiiv and ManyChat connectors; posting, uploads and DM replies go only
// through the connectors' guarded writes, refused while FOUNDEROS_WRITES=0.

func init() { RegisterPage(socialRoutes) }

const socialWorkspace = "personal"

type socialZernio interface {
	LiveAccounts(ctx context.Context) (zernio.Accounts, error)
	ConfigAccounts() zernio.Accounts
	PostDays(ctx context.Context) ([]zernio.PostDay, error)
	RecentPosts(ctx context.Context, limit int) ([]zernio.Post, error)
	UploadTarget(ctx context.Context, fileName, contentType string) (zernio.UploadSlot, error)
	Publish(ctx context.Context, in zernio.PublishInput) error
}

type socialBeehiiv interface {
	Reading(ctx context.Context) *beehiiv.Reading
	Posts(ctx context.Context) []beehiiv.Newsletter
}

type socialManyChat interface {
	SendText(ctx context.Context, subscriberID, text string) (manychat.SendResult, error)
}

type socialSources struct {
	zernio   socialZernio
	beehiiv  socialBeehiiv
	manychat socialManyChat
	http     *http.Client // the media PUT, on the guarded transport
}

// socialNewSources builds the live connectors once per server (their caches
// live on the connector). Tests swap it.
var socialNewSources = func(d *Deps) socialSources {
	return socialSources{
		zernio:   zernio.New(d.Resolver),
		beehiiv:  depsBeehiiv(d),
		manychat: manychat.New(d.Resolver),
		http:     &http.Client{Transport: guard.Transport(http.DefaultTransport), Timeout: 120 * time.Second},
	}
}

func socialToday() string { return time.Now().UTC().Format("2006-01-02") }

func socialErr(c *gin.Context, code int, err error) {
	c.JSON(code, gin.H{"error": err.Error()})
}

// socialStore resolves the personal workspace; without it the page
// answers 503 rather than reading as empty.
func socialStore(c *gin.Context, d *Deps) (*social.Store, bool) {
	ws, err := pagekit.WorkspaceID(c.Request.Context(), d.Pool, socialWorkspace)
	if err != nil {
		socialErr(c, http.StatusServiceUnavailable, err)
		return nil, false
	}
	return &social.Store{Pool: d.Pool, WorkspaceID: ws}, true
}

type socialSyncResult struct {
	Source   string           `json:"source"` // zernio-live | zernio-config | none
	Recorded int              `json:"recorded"`
	Accounts []zernio.Account `json:"accounts"`
	Error    string           `json:"error,omitempty"`
}

// socialSync is syncFromZernioLive: today's live follower counts, else the
// static config, snapshotted into Postgres. A failure never blanks a page;
// it is reported in the result.
func socialSync(ctx context.Context, src socialSources, st *social.Store) socialSyncResult {
	res := socialSyncResult{Source: "none", Accounts: []zernio.Account{}}
	live, err := src.zernio.LiveAccounts(ctx)
	accounts := live
	if err == nil && len(live) > 0 {
		res.Source = "zernio-live"
	} else {
		if err != nil {
			res.Error = "zernio: " + err.Error()
		}
		accounts = src.zernio.ConfigAccounts()
		if len(accounts) > 0 {
			res.Source = "zernio-config"
		}
	}
	if len(accounts) > 0 {
		res.Accounts = accounts
	}
	var rows []social.LiveAccount
	for _, a := range accounts {
		rows = append(rows, social.LiveAccount{Platform: a.Platform, Handle: a.Handle, Followers: a.Followers})
	}
	snaps := social.SyncRows(rows, socialToday(), res.Source)
	if err := st.InsertSnapshots(ctx, snaps); err != nil {
		res.Error = strings.TrimPrefix(res.Error+"; snapshot write: "+err.Error(), "; ")
		return res
	}
	res.Recorded = len(snaps)
	return res
}

func socialPostDays(days []zernio.PostDay) []social.PostDay {
	out := make([]social.PostDay, 0, len(days))
	for _, d := range days {
		out = append(out, social.PostDay{Date: d.Date, Platforms: d.Platforms})
	}
	return out
}

func socialRoutes(s *gin.RouterGroup, d *Deps) {
	src := socialNewSources(d)

	s.GET("/pages/social", func(c *gin.Context) {
		ctx := c.Request.Context()
		st, ok := socialStore(c, d)
		if !ok {
			return
		}
		sync := socialSync(ctx, src, st)
		x, err := st.Load(ctx)
		if err != nil {
			socialErr(c, http.StatusInternalServerError, err)
			return
		}
		today := socialToday()
		dash := social.BuildDashboard(x)
		email := social.BuildEmailList(x)

		var leader *string
		var best float64
		for _, p := range dash.Platforms {
			if p.Growth.D7 == nil {
				continue
			}
			if leader == nil || best < *p.Growth.D7 {
				name := social.Label(p.Platform)
				leader, best = &name, *p.Growth.D7
			}
		}
		threads := social.DMThreads(x, "instagram")
		queued := 0
		for _, p := range x.Posts {
			if p.Status == "queued" {
				queued++
			}
		}

		body := gin.H{}
		var postDays []social.PostDay
		if days, err := src.zernio.PostDays(ctx); err != nil {
			body["postDays"], body["postsKnown"], body["postsError"] = nil, false, err.Error()
		} else {
			postDays = socialPostDays(days)
			body["postDays"], body["postsKnown"] = postDays, true
		}
		if recent, err := src.zernio.RecentPosts(ctx, 5); err != nil {
			body["recentPosts"], body["recentError"] = nil, err.Error()
		} else {
			if recent == nil {
				recent = []zernio.Post{}
			}
			body["recentPosts"] = recent
		}

		channels := make([]social.Channel, 0, len(dash.Platforms)+1)
		for _, p := range dash.Platforms {
			channels = append(channels, social.Channel{Key: p.Platform, Label: social.Label(p.Platform), Value: p.Followers})
		}
		channels = append(channels, social.Channel{Key: "email", Label: "Email list", Value: email.Subscribers})
		ts := make([]social.ThreadState, 0, len(threads))
		for _, t := range threads {
			ts = append(ts, social.ThreadState{Name: t.Name, Unreplied: t.Unreplied, Demo: t.Demo})
		}
		total := social.AudienceTotal(x)
		growth := social.AudienceGrowth(x)
		lead := ""
		if leader != nil {
			lead = *leader
		}
		all, _ := social.AudienceSeries(x)

		for k, v := range (gin.H{
			"totalFollowers":   dash.TotalFollowers,
			"asOf":             dash.AsOf,
			"platforms":        dash.Platforms,
			"emailList":        email,
			"totalDms":         social.TotalDMs(x),
			"audienceTotal":    total,
			"audienceGrowth":   growth,
			"dmGrowth":         social.DMGrowth(x),
			"monthlyGrowthPct": growth.D30,
			"today":            today,
			"growthLeader":     leader,
			"dmThreads":        threads,
			"audiencePoints":   all.Points,
			"posts":            x.Posts,
			"queued":           queued,
			"sync":             sync,
			"volume": social.BuildSocialVolume(social.SocialVolumeInput{
				Channels: channels, Total: total, Growth7: growth.D7, Leader: lead,
				Threads: ts, Queued: queued, PostDays: postDays, Today: today,
			}),
		}) {
			body[k] = v
		}
		c.JSON(http.StatusOK, body)
	})

	s.GET("/pages/social/history", func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.Query("limit"))
		if limit == 0 {
			limit = 6
		}
		limit = min(max(limit, 1), 24)
		posts, err := src.zernio.RecentPosts(c.Request.Context(), limit)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"posts": nil, "error": "zernio history unavailable: " + err.Error()})
			return
		}
		if posts == nil {
			posts = []zernio.Post{}
		}
		c.JSON(http.StatusOK, gin.H{"posts": posts})
	})

	s.GET("/pages/social/series", func(c *gin.Context) {
		st, ok := socialStore(c, d)
		if !ok {
			return
		}
		x, err := st.Load(c.Request.Context())
		if err != nil {
			socialErr(c, http.StatusInternalServerError, err)
			return
		}
		ranges := []any{7, 30, 60, "all"}
		switch metric := c.Query("metric"); metric {
		case "audience":
			all, channels := social.AudienceSeries(x)
			c.JSON(http.StatusOK, gin.H{"metric": metric, "ranges": ranges, "series": append([]social.LabelledSeries{all}, channels...)})
		case "dms":
			c.JSON(http.StatusOK, gin.H{"metric": metric, "ranges": ranges, "series": []social.LabelledSeries{
				{Key: "total", Label: "Total DMs", Color: social.DMColor, Points: social.DMSeries(x)},
			}})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "metric must be 'audience' or 'dms'"})
		}
	})

	runSync := func(c *gin.Context) {
		st, ok := socialStore(c, d)
		if !ok {
			return
		}
		r := socialSync(c.Request.Context(), src, st)
		c.JSON(http.StatusOK, gin.H{"ok": r.Error == "" || r.Recorded > 0, "recorded": r.Recorded, "syncedAt": time.Now().UTC().Format(time.RFC3339),
			"source": r.Source, "accounts": r.Accounts, "error": r.Error})
	}
	s.GET("/pages/social/sync", runSync)
	s.POST("/pages/social/sync", runSync)

	s.POST("/pages/social/upload", func(c *gin.Context) {
		fh, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": `multipart "file" field required`})
			return
		}
		ctype := fh.Header.Get("Content-Type")
		if ctype == "" {
			ctype = "application/octet-stream"
		}
		slot, err := src.zernio.UploadTarget(c.Request.Context(), fh.Filename, ctype)
		if err != nil {
			socialErr(c, http.StatusBadGateway, err)
			return
		}
		f, err := fh.Open()
		if err != nil {
			socialErr(c, http.StatusBadRequest, err)
			return
		}
		defer f.Close()
		raw, err := io.ReadAll(f)
		if err != nil {
			socialErr(c, http.StatusBadRequest, err)
			return
		}
		req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPut, slot.UploadURL, bytes.NewReader(raw))
		if err != nil {
			socialErr(c, http.StatusBadGateway, err)
			return
		}
		req.Header.Set("Content-Type", ctype)
		res, err := src.http.Do(req)
		if err != nil {
			socialErr(c, http.StatusBadGateway, err)
			return
		}
		res.Body.Close()
		if res.StatusCode < 200 || res.StatusCode > 299 {
			socialErr(c, http.StatusBadGateway, fmt.Errorf("media PUT failed: HTTP %d", res.StatusCode))
			return
		}
		c.JSON(http.StatusCreated, gin.H{"url": slot.AccessURL, "name": fh.Filename, "contentType": ctype})
	})

	s.GET("/pages/social/posts", func(c *gin.Context) {
		st, ok := socialStore(c, d)
		if !ok {
			return
		}
		posts, err := st.Posts(c.Request.Context())
		if err != nil {
			socialErr(c, http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"posts": posts})
	})

	s.POST("/pages/social/posts", func(c *gin.Context) {
		var in struct {
			Caption      string   `json:"caption"`
			Platforms    []string `json:"platforms"`
			MediaURL     *string  `json:"mediaUrl"`
			MediaURLs    []string `json:"mediaUrls"`
			ScheduledFor *string  `json:"scheduledFor"`
		}
		if err := c.ShouldBindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "body must be JSON"})
			return
		}
		if msg := socialValidatePost(in.Caption, in.Platforms, in.MediaURL, in.MediaURLs); msg != "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": msg})
			return
		}
		st, ok := socialStore(c, d)
		if !ok {
			return
		}
		media := in.MediaURLs
		if len(media) == 0 && in.MediaURL != nil && *in.MediaURL != "" {
			media = []string{*in.MediaURL}
		}
		sched := in.ScheduledFor
		if sched != nil && *sched == "" {
			sched = nil
		}
		status, publishErr := "published", ""
		if err := src.zernio.Publish(c.Request.Context(), zernio.PublishInput{Caption: in.Caption, MediaURLs: media, Platforms: in.Platforms, ScheduledFor: sched}); err != nil {
			status, publishErr = "failed", err.Error()
		} else if sched != nil {
			status = "queued"
		}
		post := social.Post{ID: uuid.NewString(), Caption: in.Caption, Platforms: in.Platforms, Status: status, ScheduledFor: sched,
			CreatedAt: time.Now().UTC().Format("2006-01-02T15:04:05.000Z")}
		if len(media) > 0 {
			post.MediaURL = &media[0]
		}
		if err := st.EnqueuePost(c.Request.Context(), post); err != nil {
			socialErr(c, http.StatusInternalServerError, err)
			return
		}
		if publishErr != "" {
			c.JSON(http.StatusBadGateway, gin.H{"post": post, "error": publishErr})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"post": post})
	})

	s.POST("/pages/social/dm/reply", func(c *gin.Context) {
		var in struct {
			SubscriberID string `json:"subscriberId"`
			Text         string `json:"text"`
		}
		if err := c.ShouldBindJSON(&in); err != nil || in.SubscriberID == "" || in.Text == "" || len([]rune(in.Text)) > 2000 {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "subscriberId and text (1-2000 chars) are required"})
			return
		}
		st, ok := socialStore(c, d)
		if !ok {
			return
		}
		res, err := src.manychat.SendText(c.Request.Context(), in.SubscriberID, in.Text)
		if err != nil || !res.OK {
			detail := res.Detail
			if detail == "" && err != nil {
				detail = err.Error()
			}
			c.JSON(http.StatusBadGateway, gin.H{"ok": false, "error": detail})
			return
		}
		msg := social.DMMessage{Platform: "instagram", SubscriberID: in.SubscriberID, Name: in.SubscriberID, Text: in.Text, Direction: "out", Source: "manychat",
			TS: time.Now().UTC().Format("2006-01-02T15:04:05.000Z")}
		msg.ID = "mc-out-" + in.SubscriberID + "-" + msg.TS
		if prior, err := st.DMMessages(c.Request.Context()); err == nil {
			for _, m := range prior {
				if m.Platform == "instagram" && m.SubscriberID == in.SubscriberID {
					msg.Name, msg.Handle = m.Name, m.Handle
					break
				}
			}
		}
		// The DM went out; a failed local write must not read as a failed send.
		if err := st.UpsertDMMessage(c.Request.Context(), msg); err != nil {
			c.JSON(http.StatusOK, gin.H{"ok": true, "message": msg, "warning": "sent, but not stored: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "message": msg})
	})

	s.GET("/pages/social/beehiiv", func(c *gin.Context) {
		ctx := c.Request.Context()
		reading := src.beehiiv.Reading(ctx)
		posts := src.beehiiv.Posts(ctx)
		seeded := len(posts) == 0
		if seeded {
			posts = social.SeedNewsletters
		}
		var subs *int
		fresh := false
		if reading != nil {
			n := reading.Subscribers
			subs, fresh = &n, reading.Fresh
		}
		c.JSON(http.StatusOK, gin.H{"newsletters": posts, "seeded": seeded, "subscribers": subs, "live": reading != nil, "fresh": fresh,
			"volume": social.BuildNewsletterVolume(posts, subs)})
	})

	// Registered after the static siblings above; gin resolves static
	// segments first either way (see TestSocialStaticRoutesBeatThePlatformParam).
	s.GET("/pages/social/:platform", func(c *gin.Context) {
		st, ok := socialStore(c, d)
		if !ok {
			return
		}
		platform := c.Param("platform")
		if !social.Tracked(platform) {
			c.JSON(http.StatusNotFound, gin.H{"error": "unknown platform: " + platform})
			return
		}
		sync := socialSync(c.Request.Context(), src, st)
		x, err := st.Load(c.Request.Context())
		if err != nil {
			socialErr(c, http.StatusInternalServerError, err)
			return
		}
		detail := social.PlatformDetail(x, platform)
		if detail == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "unknown platform: " + platform})
			return
		}
		today := socialToday()
		label := social.Label(platform)
		c.JSON(http.StatusOK, gin.H{
			"account": detail.Account, "followers": detail.Followers, "growth": detail.Growth, "snapshots": detail.Snapshots,
			"label": label, "today": today, "sync": sync,
			"volume": social.BuildPlatformVolume(label, detail.Followers, detail.Growth, detail.Snapshots, today, 30),
		})
	})
}

// socialValidatePost is the TS CreateSchema.
func socialValidatePost(caption string, platforms []string, mediaURL *string, mediaURLs []string) string {
	if caption == "" {
		return "caption is required"
	}
	if len(platforms) == 0 {
		return "pick at least one platform"
	}
	for _, p := range platforms {
		if !social.Tracked(p) {
			return "unknown platform: " + p
		}
	}
	urls := append([]string(nil), mediaURLs...)
	if mediaURL != nil && *mediaURL != "" {
		urls = append(urls, *mediaURL)
	}
	for _, u := range urls {
		if pu, err := url.Parse(u); err != nil || pu.Scheme == "" || pu.Host == "" {
			return "media URL must be a URL: " + u
		}
	}
	return ""
}
