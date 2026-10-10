package api

import (
	"sync"

	"github.com/gin-gonic/gin"
)

// PageRoute mounts one page's data endpoints on the session-authed
// /api/founderos group. Page ports register from their own file's init(), so
// parallel ports never edit a shared registration list:
//
//	func init() { RegisterPage(registerFunnel) }
type PageRoute func(s *gin.RouterGroup, d *Deps)

var (
	pagesMu sync.Mutex
	pages   []PageRoute
)

func RegisterPage(r PageRoute) {
	pagesMu.Lock()
	defer pagesMu.Unlock()
	pages = append(pages, r)
}

func mountPages(s *gin.RouterGroup, d *Deps) {
	pagesMu.Lock()
	defer pagesMu.Unlock()
	for _, p := range pages {
		p(s, d)
	}
}
