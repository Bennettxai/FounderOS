// Package microsoft provides Microsoft 365 integration (Outlook, OneDrive, Teams, etc.).
package microsoft

const (
	ProviderID   = "microsoft"
	ProviderName = "Microsoft 365"
	Category     = "productivity"

	// Microsoft OAuth endpoints
	AuthURL  = "https://login.microsoftonline.com/common/oauth2/v2.0/authorize"
	TokenURL = "https://login.microsoftonline.com/common/oauth2/v2.0/token"

	// Microsoft Graph API base URL
	GraphAPIBase = "https://graph.microsoft.com/v1.0"
)

// Microsoft Graph API Scopes - COMPREHENSIVE ACCESS
var (
	// User profile
	UserScopes = []string{
		"User.Read",              // Read user profile
		"User.ReadBasic.All",     // Read basic profiles of all users
		"User.Read.All",          // Read all users' full profiles
		"User.ReadWrite",         // Read and update user profile
	}

	// Mail/Outlook
	MailScopes = []string{
		"Mail.Read",              // Read user mail
		"Mail.ReadBasic",         // Read user mail (basic)
		"Mail.ReadWrite",         // Read and write user mail
		"Mail.Send",              // Send mail as user
		"MailboxSettings.Read",   // Read mailbox settings
		"MailboxSettings.ReadWrite", // Read and write mailbox settings
	}

	// Calendar
	CalendarScopes = []string{
		"Calendars.Read",         // Read user calendars
		"Calendars.Read.Shared",  // Read shared calendars
		"Calendars.ReadWrite",    // Read and write user calendars
		"Calendars.ReadWrite.Shared", // Read and write shared calendars
	}

	// Contacts
	ContactsScopes = []string{
		"Contacts.Read",          // Read user contacts
		"Contacts.Read.Shared",   // Read shared contacts
		"Contacts.ReadWrite",     // Read and write user contacts
		"Contacts.ReadWrite.Shared", // Read and write shared contacts
	}

	// OneDrive/Files
	FilesScopes = []string{
		"Files.Read",             // Read user files
		"Files.Read.All",         // Read all files user can access
		"Files.ReadWrite",        // Read and write user files
		"Files.ReadWrite.All",    // Read and write all files user can access
		"Files.Read.Selected",    // Read files selected by user
		"Files.ReadWrite.Selected", // Read and write files selected by user
	}

	// To Do Tasks
	TasksScopes = []string{
		"Tasks.Read",             // Read user tasks
		"Tasks.Read.Shared",      // Read shared tasks
		"Tasks.ReadWrite",        // Read and write user tasks
		"Tasks.ReadWrite.Shared", // Read and write shared tasks
	}

	// OneNote
	OneNoteScopes = []string{
		"Notes.Read",             // Read OneNote notebooks
		"Notes.Read.All",         // Read all OneNote notebooks
		"Notes.ReadWrite",        // Read and write OneNote notebooks
		"Notes.ReadWrite.All",    // Read and write all OneNote notebooks
		"Notes.Create",           // Create OneNote notebooks
	}

	// Teams
	TeamsScopes = []string{
		"Team.ReadBasic.All",     // Read basic team info
		"TeamSettings.Read.All",  // Read team settings
		"TeamSettings.ReadWrite.All", // Read and write team settings
		"Channel.ReadBasic.All",  // Read channel basic info
		"ChannelMessage.Read.All", // Read channel messages
		"ChannelMessage.Send",    // Send channel messages
		"Chat.Read",              // Read chat messages
		"Chat.ReadWrite",         // Read and write chat messages
		"ChatMessage.Read",       // Read chat messages
		"ChatMessage.Send",       // Send chat messages
	}

	// SharePoint/Sites
	SitesScopes = []string{
		"Sites.Read.All",         // Read all site collections
		"Sites.ReadWrite.All",    // Read and write all site collections
		"Sites.Manage.All",       // Create, edit, delete site collections
		"Sites.FullControl.All",  // Full control of all site collections
	}

	// Groups
	GroupsScopes = []string{
		"Group.Read.All",         // Read all groups
		"Group.ReadWrite.All",    // Read and write all groups
		"GroupMember.Read.All",   // Read group members
		"GroupMember.ReadWrite.All", // Read and write group members
	}

	// Planner
	PlannerScopes = []string{
		"Tasks.Read",             // Read Planner tasks (shared with To Do)
		"Tasks.ReadWrite",        // Read and write Planner tasks
		"Group.Read.All",         // Required for Planner groups
	}

	// Directory
	DirectoryScopes = []string{
		"Directory.Read.All",     // Read directory data
		"Directory.ReadWrite.All", // Read and write directory data
		"Directory.AccessAsUser.All", // Access directory as user
	}

	// People
	PeopleScopes = []string{
		"People.Read",            // Read user's relevant people
		"People.Read.All",        // Read all users' relevant people
	}

	// Bookings
	BookingsScopes = []string{
		"Bookings.Read.All",      // Read booking businesses
		"Bookings.ReadWrite.All", // Read and write booking businesses
		"Bookings.Manage.All",    // Manage booking businesses
		"BookingsAppointment.ReadWrite.All", // Read and write appointments
	}

	// Reports
	ReportsScopes = []string{
		"Reports.Read.All",       // Read all usage reports
	}

	// Security
	SecurityScopes = []string{
		"SecurityEvents.Read.All", // Read security events
		"SecurityEvents.ReadWrite.All", // Read and write security events
	}

	// Audit logs
	AuditScopes = []string{
		"AuditLog.Read.All",      // Read audit logs
	}

	// Offline access (for refresh tokens)
	OfflineScopes = []string{
		"offline_access",         // Get refresh tokens
	}

	// OpenID Connect
	OpenIDScopes = []string{
		"openid",
		"profile",
		"email",
	}

	// AllMicrosoftScopes contains all available Microsoft scopes
	AllMicrosoftScopes []string
)

func init() {
	// Build the complete list of all scopes
	AllMicrosoftScopes = make([]string, 0)
	AllMicrosoftScopes = append(AllMicrosoftScopes, OpenIDScopes...)
	AllMicrosoftScopes = append(AllMicrosoftScopes, OfflineScopes...)
	AllMicrosoftScopes = append(AllMicrosoftScopes, UserScopes...)
	AllMicrosoftScopes = append(AllMicrosoftScopes, MailScopes...)
	AllMicrosoftScopes = append(AllMicrosoftScopes, CalendarScopes...)
	AllMicrosoftScopes = append(AllMicrosoftScopes, ContactsScopes...)
	AllMicrosoftScopes = append(AllMicrosoftScopes, FilesScopes...)
	AllMicrosoftScopes = append(AllMicrosoftScopes, TasksScopes...)
	AllMicrosoftScopes = append(AllMicrosoftScopes, OneNoteScopes...)
	AllMicrosoftScopes = append(AllMicrosoftScopes, TeamsScopes...)
	AllMicrosoftScopes = append(AllMicrosoftScopes, SitesScopes...)
	AllMicrosoftScopes = append(AllMicrosoftScopes, GroupsScopes...)
	AllMicrosoftScopes = append(AllMicrosoftScopes, PlannerScopes...)
	AllMicrosoftScopes = append(AllMicrosoftScopes, DirectoryScopes...)
	AllMicrosoftScopes = append(AllMicrosoftScopes, PeopleScopes...)
	AllMicrosoftScopes = append(AllMicrosoftScopes, BookingsScopes...)
	AllMicrosoftScopes = append(AllMicrosoftScopes, ReportsScopes...)
	AllMicrosoftScopes = append(AllMicrosoftScopes, SecurityScopes...)
	AllMicrosoftScopes = append(AllMicrosoftScopes, AuditScopes...)
}
