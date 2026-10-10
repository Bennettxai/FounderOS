// Package google provides the Google Workspace integration (Calendar, Gmail, Drive).
package google

import (
	googlecalendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/gmail/v1"
)

const (
	ProviderID   = "google"
	ProviderName = "Google Workspace"
	Category     = "productivity"
)

// Scopes for different Google services - MINIMAL REQUIRED SCOPES
// Only request what's needed and what's enabled in Google Cloud Console
var (
	// Calendar: Basic calendar access (read events, create/modify events)
	CalendarScopes = []string{
		googlecalendar.CalendarReadonlyScope, // Read calendar list and events
		googlecalendar.CalendarEventsScope,   // Create/edit/delete events
	}

	// Gmail: Basic email access (read, send, modify)
	// NOTE: gmail.readonly scope is now requested during OAuth in auth_google.go
	// Frontend should handle cases where user denies Gmail access
	// Check: internal/integrations/google/gmail.go IsConnected() before syncing
	GmailScopes = []string{
		gmail.GmailReadonlyScope, // Read emails (required for SyncEmails)
		gmail.GmailSendScope,     // Send emails
		gmail.GmailModifyScope,   // Modify emails (archive, delete, labels)
	}

	// Drive: Basic file access
	DriveScopes = []string{
		"https://www.googleapis.com/auth/drive.readonly", // Read files and metadata
		"https://www.googleapis.com/auth/drive.file",     // Create/edit files created by app
	}

	// Contacts/People: Basic contacts access
	ContactsScopes = []string{
		"https://www.googleapis.com/auth/contacts.readonly", // Read contacts
	}

	// Tasks: Basic tasks access
	TasksScopes = []string{
		"https://www.googleapis.com/auth/tasks", // Create, edit, delete tasks
	}

	// Sheets: Basic spreadsheet access
	SheetsScopes = []string{
		"https://www.googleapis.com/auth/spreadsheets.readonly", // Read spreadsheets
	}

	// Docs: Basic document access
	DocsScopes = []string{
		"https://www.googleapis.com/auth/documents.readonly", // Read documents
	}

	// Slides: Basic presentation access
	SlidesScopes = []string{
		"https://www.googleapis.com/auth/presentations.readonly", // Read presentations
	}

	// Forms: Basic form access
	FormsScopes = []string{
		"https://www.googleapis.com/auth/forms.currentonly", // Manage forms the app is installed in
	}

	// Chat: Basic chat access (requires Chat API enabled)
	ChatScopes = []string{
		"https://www.googleapis.com/auth/chat.messages.readonly", // Read messages
	}

	// Photos: Basic photos access (requires Photos API enabled)
	PhotosScopes = []string{
		"https://www.googleapis.com/auth/photoslibrary.readonly", // View Google Photos
	}

	// YouTube: Basic YouTube access (requires YouTube API enabled)
	YouTubeScopes = []string{
		"https://www.googleapis.com/auth/youtube.readonly", // View YouTube account
	}

	// Blogger: Basic blog access (requires Blogger API enabled)
	BloggerScopes = []string{
		"https://www.googleapis.com/auth/blogger.readonly", // View Blogger account
	}

	// Classroom: Basic classroom access (requires Classroom API enabled)
	ClassroomScopes = []string{
		"https://www.googleapis.com/auth/classroom.courses.readonly", // View classes
	}

	// User info: Basic profile information
	UserInfoScopes = []string{
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/userinfo.profile",
		"openid",
	}

	// Cloud Platform: GCP access (requires Cloud Platform API)
	CloudScopes = []string{
		"https://www.googleapis.com/auth/cloud-platform.read-only", // Read GCP access
	}

	// Meet: Google Meet access (requires Meet API)
	MeetScopes = []string{
		"https://www.googleapis.com/auth/meetings.space.readonly", // View meeting space info
	}

	// Keep: Google Keep notes access (requires Keep API)
	KeepScopes = []string{
		"https://www.googleapis.com/auth/keep.readonly", // Read Keep notes
	}

	// Analytics: Google Analytics access (requires Analytics API)
	AnalyticsScopes = []string{
		"https://www.googleapis.com/auth/analytics.readonly", // Read Analytics data
	}

	// Admin SDK: Google Workspace Admin access (requires Admin SDK API + admin account)
	AdminScopes = []string{
		"https://www.googleapis.com/auth/admin.directory.user.readonly", // View users
	}

	// Fitness: Google Fit access (requires Fitness API)
	FitnessScopes = []string{
		"https://www.googleapis.com/auth/fitness.activity.read", // Read activity data
	}

	// Ads: Google Ads access (requires Google Ads API)
	AdsScopes = []string{
		"https://www.googleapis.com/auth/adwords", // Google Ads access
	}

	// Search Console: Google Search Console access (requires Search Console API)
	SearchConsoleScopes = []string{
		"https://www.googleapis.com/auth/webmasters.readonly", // Read Search Console data
	}

	// BigQuery: BigQuery access (requires BigQuery API)
	BigQueryScopes = []string{
		"https://www.googleapis.com/auth/bigquery.readonly", // Read BigQuery data
	}

	// Pub/Sub: Google Pub/Sub access (requires Pub/Sub API)
	PubSubScopes = []string{
		"https://www.googleapis.com/auth/pubsub", // Pub/Sub access
	}

	// Storage: Google Cloud Storage access (requires Cloud Storage API)
	StorageScopes = []string{
		"https://www.googleapis.com/auth/devstorage.read_only", // Read GCS objects
	}

	// AllGoogleScopes contains EVERY Google scope for maximum access
	AllGoogleScopes []string
)

func init() {
	// Build the complete list of all scopes
	AllGoogleScopes = make([]string, 0)
	AllGoogleScopes = append(AllGoogleScopes, UserInfoScopes...)
	AllGoogleScopes = append(AllGoogleScopes, CalendarScopes...)
	AllGoogleScopes = append(AllGoogleScopes, GmailScopes...)
	AllGoogleScopes = append(AllGoogleScopes, DriveScopes...)
	AllGoogleScopes = append(AllGoogleScopes, ContactsScopes...)
	AllGoogleScopes = append(AllGoogleScopes, TasksScopes...)
	AllGoogleScopes = append(AllGoogleScopes, SheetsScopes...)
	AllGoogleScopes = append(AllGoogleScopes, DocsScopes...)
	AllGoogleScopes = append(AllGoogleScopes, SlidesScopes...)
	AllGoogleScopes = append(AllGoogleScopes, FormsScopes...)
	AllGoogleScopes = append(AllGoogleScopes, ChatScopes...)
	AllGoogleScopes = append(AllGoogleScopes, PhotosScopes...)
	AllGoogleScopes = append(AllGoogleScopes, YouTubeScopes...)
	AllGoogleScopes = append(AllGoogleScopes, BloggerScopes...)
	AllGoogleScopes = append(AllGoogleScopes, ClassroomScopes...)
	AllGoogleScopes = append(AllGoogleScopes, CloudScopes...)
	AllGoogleScopes = append(AllGoogleScopes, MeetScopes...)
	AllGoogleScopes = append(AllGoogleScopes, KeepScopes...)
	AllGoogleScopes = append(AllGoogleScopes, AnalyticsScopes...)
	AllGoogleScopes = append(AllGoogleScopes, AdminScopes...)
	AllGoogleScopes = append(AllGoogleScopes, FitnessScopes...)
	AllGoogleScopes = append(AllGoogleScopes, AdsScopes...)
	AllGoogleScopes = append(AllGoogleScopes, SearchConsoleScopes...)
	AllGoogleScopes = append(AllGoogleScopes, BigQueryScopes...)
	AllGoogleScopes = append(AllGoogleScopes, PubSubScopes...)
	AllGoogleScopes = append(AllGoogleScopes, StorageScopes...)
}
