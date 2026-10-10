package clickup

// ============================================================================
// ClickUp API Types
// ============================================================================

// Workspace represents a ClickUp workspace (called Team in API v2)
type Workspace struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Color   string   `json:"color"`
	Avatar  string   `json:"avatar"`
	Members []Member `json:"members"`
}

// Member represents a workspace member
type Member struct {
	User struct {
		ID             int    `json:"id"`
		Username       string `json:"username"`
		Email          string `json:"email"`
		Color          string `json:"color"`
		ProfilePicture string `json:"profilePicture"`
		Initials       string `json:"initials"`
	} `json:"user"`
}

// Space represents a ClickUp space
type Space struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Private  bool   `json:"private"`
	Color    string `json:"color"`
	Avatar   string `json:"avatar"`
	Archived bool   `json:"archived"`
}

// Folder represents a ClickUp folder
type Folder struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Hidden   bool   `json:"hidden"`
	Archived bool   `json:"archived"`
	Space    struct {
		ID string `json:"id"`
	} `json:"space"`
}

// List represents a ClickUp list
type List struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Archived bool   `json:"archived"`
	Folder   struct {
		ID string `json:"id"`
	} `json:"folder"`
	Space struct {
		ID string `json:"id"`
	} `json:"space"`
	TaskCount int `json:"task_count"`
}

// Task represents a ClickUp task
type Task struct {
	ID          string   `json:"id"`
	CustomID    string   `json:"custom_id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Status      Status   `json:"status"`
	Priority    Priority `json:"priority"`
	DueDate     string   `json:"due_date"`
	StartDate   string   `json:"start_date"`
	TimeSpent   int64    `json:"time_spent"`
	Assignees   []struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
	} `json:"assignees"`
	Creator struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
	} `json:"creator"`
	Tags []struct {
		Name string `json:"name"`
	} `json:"tags"`
	Parent      string     `json:"parent"`
	Subtasks    []Task     `json:"subtasks"`
	DateCreated string     `json:"date_created"`
	DateUpdated string     `json:"date_updated"`
	DateClosed  string     `json:"date_closed"`
	URL         string     `json:"url"`
	List        ListRef    `json:"list"`
	Folder      FolderRef  `json:"folder"`
	Space       SpaceRef   `json:"space"`
}

// Status represents a task status
type Status struct {
	Status     string `json:"status"`
	Color      string `json:"color"`
	Type       string `json:"type"`
	Orderindex int    `json:"orderindex"`
}

// Priority represents a task priority
type Priority struct {
	Priority string `json:"priority"`
	Color    string `json:"color"`
}

// ListRef represents a reference to a list
type ListRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// FolderRef represents a reference to a folder
type FolderRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// SpaceRef represents a reference to a space
type SpaceRef struct {
	ID string `json:"id"`
}

// clickUpUser represents the authenticated user
type clickUpUser struct {
	ID             int    `json:"id"`
	Username       string `json:"username"`
	Email          string `json:"email"`
	Color          string `json:"color"`
	Initials       string `json:"initials"`
	ProfilePicture string `json:"profilePicture"`
}
