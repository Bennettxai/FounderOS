// Package handlers provides HTTP handlers for the BusinessOS API.
// OAuth integration handlers are split across:
//   - oauth_common.go    — OAuthIntegrationHandler struct, constructor, shared helpers
//   - oauth_slack.go     — Slack OAuth initiation and callback
//   - oauth_microsoft.go — Microsoft (Outlook) OAuth initiation and callback
//   - oauth_hubspot.go   — Notion and Linear OAuth initiation and callbacks
package handlers
