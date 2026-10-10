package handlers

import "net/http"

// sessionCookieSameSite supports both the same-origin web client and the
// packaged app:// Electron renderer. CSRF protection is enforced separately by
// the double-submit cookie and explicit header middleware.
func sessionCookieSameSite(isProduction bool) http.SameSite {
	if isProduction {
		return http.SameSiteNoneMode
	}
	return http.SameSiteLaxMode
}
