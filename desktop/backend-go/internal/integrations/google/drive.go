package google

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// DriveFile represents a synced Google Drive file.
type DriveFile struct {
	ID               string            `json:"id"`
	UserID           string            `json:"user_id"`
	FileID           string            `json:"file_id"`
	Name             string            `json:"name"`
	MimeType         string            `json:"mime_type,omitempty"`
	FileExtension    string            `json:"file_extension,omitempty"`
	SizeBytes        int64             `json:"size_bytes,omitempty"`
	ParentFolderID   string            `json:"parent_folder_id,omitempty"`
	ParentFolderName string            `json:"parent_folder_name,omitempty"`
	Path             string            `json:"path,omitempty"`
	Shared           bool              `json:"shared"`
	SharingUser      string            `json:"sharing_user,omitempty"`
	Permissions      []FilePermission  `json:"permissions,omitempty"`
	WebViewLink      string            `json:"web_view_link,omitempty"`
	WebContentLink   string            `json:"web_content_link,omitempty"`
	ThumbnailLink    string            `json:"thumbnail_link,omitempty"`
	IconLink         string            `json:"icon_link,omitempty"`
	CreatedTime      time.Time         `json:"created_time,omitempty"`
	ModifiedTime     time.Time         `json:"modified_time,omitempty"`
	ViewedByMeTime   time.Time         `json:"viewed_by_me_time,omitempty"`
	Owners           []FileOwner       `json:"owners,omitempty"`
	LastModifyingUser *FileOwner       `json:"last_modifying_user,omitempty"`
	SyncedAt         time.Time         `json:"synced_at"`
}

// FilePermission represents a permission on a file.
type FilePermission struct {
	ID          string `json:"id"`
	Type        string `json:"type"`        // user, group, domain, anyone
	Role        string `json:"role"`        // owner, organizer, fileOrganizer, writer, commenter, reader
	EmailAddr   string `json:"email_address,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
}

// FileOwner represents an owner or last modifier of a file.
type FileOwner struct {
	DisplayName string `json:"display_name"`
	EmailAddr   string `json:"email_address"`
	PhotoLink   string `json:"photo_link,omitempty"`
}

// DriveService handles Google Drive operations.
type DriveService struct {
	provider *Provider
}

// NewDriveService creates a new Drive service.
func NewDriveService(provider *Provider) *DriveService {
	return &DriveService{provider: provider}
}

// GetDriveAPI returns a Google Drive API service for a user.
func (s *DriveService) GetDriveAPI(ctx context.Context, userID string) (*drive.Service, error) {
	tokenSource, err := s.provider.GetTokenSource(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get token source: %w", err)
	}

	srv, err := drive.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, fmt.Errorf("failed to create drive service: %w", err)
	}

	return srv, nil
}

// SyncFiles syncs files from Google Drive.
func (s *DriveService) SyncFiles(ctx context.Context, userID string, maxResults int64) (*SyncFilesResult, error) {
	slog.Info("Drive sync starting for user : max  files", "user_id", userID, "max_results", maxResults)

	srv, err := s.GetDriveAPI(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Drive API: %w", err)
	}

	// List files from Drive
	req := srv.Files.List().
		PageSize(maxResults).
		Fields("nextPageToken, files(id, name, mimeType, fileExtension, size, parents, shared, sharingUser, webViewLink, webContentLink, thumbnailLink, iconLink, createdTime, modifiedTime, viewedByMeTime, owners, lastModifyingUser, permissions)").
		OrderBy("modifiedTime desc")

	result := &SyncFilesResult{}
	pageToken := ""

	for {
		if pageToken != "" {
			req.PageToken(pageToken)
		}

		files, err := req.Do()
		if err != nil {
			return nil, fmt.Errorf("failed to list files: %w", err)
		}

		result.TotalFiles += len(files.Files)

		for _, file := range files.Files {
			if err := s.saveFile(ctx, userID, file); err != nil {
				slog.Info("Failed to save file", "id", file.Id, "error", err)
				result.FailedFiles++
			} else {
				result.SyncedFiles++
			}
		}

		pageToken = files.NextPageToken
		if pageToken == "" || int64(result.TotalFiles) >= maxResults {
			break
		}
	}

	slog.Info("Drive sync complete for user : synced / files", "user_id", userID, "synced", result.SyncedFiles, "total", result.TotalFiles)

	return result, nil
}

// SyncFilesResult represents the result of a file sync.
type SyncFilesResult struct {
	TotalFiles  int `json:"total_files"`
	SyncedFiles int `json:"synced_files"`
	FailedFiles int `json:"failed_files"`
}

// saveFile saves a Google Drive file to the database.
func (s *DriveService) saveFile(ctx context.Context, userID string, file *drive.File) error {
	// Parse created time
	var createdTime, modifiedTime, viewedByMeTime *time.Time
	if file.CreatedTime != "" {
		t, _ := time.Parse(time.RFC3339, file.CreatedTime)
		createdTime = &t
	}
	if file.ModifiedTime != "" {
		t, _ := time.Parse(time.RFC3339, file.ModifiedTime)
		modifiedTime = &t
	}
	if file.ViewedByMeTime != "" {
		t, _ := time.Parse(time.RFC3339, file.ViewedByMeTime)
		viewedByMeTime = &t
	}

	// Get parent folder
	var parentFolderID string
	if len(file.Parents) > 0 {
		parentFolderID = file.Parents[0]
	}

	// Extract sharing user
	var sharingUser string
	if file.SharingUser != nil {
		sharingUser = file.SharingUser.EmailAddress
	}

	// Build owners JSON
	owners := make([]FileOwner, 0)
	for _, o := range file.Owners {
		owners = append(owners, FileOwner{
			DisplayName: o.DisplayName,
			EmailAddr:   o.EmailAddress,
			PhotoLink:   o.PhotoLink,
		})
	}

	// Build last modifying user JSON
	var lastModifyingUser *FileOwner
	if file.LastModifyingUser != nil {
		lastModifyingUser = &FileOwner{
			DisplayName: file.LastModifyingUser.DisplayName,
			EmailAddr:   file.LastModifyingUser.EmailAddress,
			PhotoLink:   file.LastModifyingUser.PhotoLink,
		}
	}

	// Build permissions JSON
	permissions := make([]FilePermission, 0)
	for _, p := range file.Permissions {
		permissions = append(permissions, FilePermission{
			ID:          p.Id,
			Type:        p.Type,
			Role:        p.Role,
			EmailAddr:   p.EmailAddress,
			DisplayName: p.DisplayName,
		})
	}

	// Insert or update file
	_, err := s.provider.Pool().Exec(ctx, `
		INSERT INTO google_drive_files (
			user_id, file_id, name, mime_type, file_extension, size_bytes,
			parent_folder_id, shared, sharing_user, permissions,
			web_view_link, web_content_link, thumbnail_link, icon_link,
			created_time, modified_time, viewed_by_me_time,
			owners, last_modifying_user, synced_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, NOW())
		ON CONFLICT (user_id, file_id) DO UPDATE SET
			name = EXCLUDED.name,
			mime_type = EXCLUDED.mime_type,
			file_extension = EXCLUDED.file_extension,
			size_bytes = EXCLUDED.size_bytes,
			parent_folder_id = EXCLUDED.parent_folder_id,
			shared = EXCLUDED.shared,
			sharing_user = EXCLUDED.sharing_user,
			permissions = EXCLUDED.permissions,
			web_view_link = EXCLUDED.web_view_link,
			web_content_link = EXCLUDED.web_content_link,
			thumbnail_link = EXCLUDED.thumbnail_link,
			icon_link = EXCLUDED.icon_link,
			modified_time = EXCLUDED.modified_time,
			viewed_by_me_time = EXCLUDED.viewed_by_me_time,
			owners = EXCLUDED.owners,
			last_modifying_user = EXCLUDED.last_modifying_user,
			synced_at = NOW(),
			updated_at = NOW()
	`, userID, file.Id, file.Name, file.MimeType, file.FileExtension, file.Size,
		parentFolderID, file.Shared, sharingUser, permissions,
		file.WebViewLink, file.WebContentLink, file.ThumbnailLink, file.IconLink,
		createdTime, modifiedTime, viewedByMeTime,
		owners, lastModifyingUser)

	return err
}

// GetFiles retrieves Drive files for a user.
func (s *DriveService) GetFiles(ctx context.Context, userID string, mimeType string, limit, offset int) ([]*DriveFile, error) {
	query := `
		SELECT id, user_id, file_id, name, mime_type, file_extension, size_bytes,
			parent_folder_id, parent_folder_name, path, shared, sharing_user,
			web_view_link, web_content_link, thumbnail_link, icon_link,
			created_time, modified_time, viewed_by_me_time, synced_at
		FROM google_drive_files
		WHERE user_id = $1
	`
	args := []interface{}{userID}

	if mimeType != "" {
		query += " AND mime_type = $2"
		args = append(args, mimeType)
		query += " ORDER BY modified_time DESC NULLS LAST LIMIT $3 OFFSET $4"
		args = append(args, limit, offset)
	} else {
		query += " ORDER BY modified_time DESC NULLS LAST LIMIT $2 OFFSET $3"
		args = append(args, limit, offset)
	}

	rows, err := s.provider.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*DriveFile
	for rows.Next() {
		var f DriveFile
		var mimeType, fileExt, parentFolderID, parentFolderName, path pgtype.Text
		var sharingUser, webViewLink, webContentLink, thumbnailLink, iconLink pgtype.Text
		var createdTime, modifiedTime, viewedByMeTime pgtype.Timestamptz
		var sizeBytes pgtype.Int8

		err := rows.Scan(
			&f.ID, &f.UserID, &f.FileID, &f.Name, &mimeType, &fileExt, &sizeBytes,
			&parentFolderID, &parentFolderName, &path, &f.Shared, &sharingUser,
			&webViewLink, &webContentLink, &thumbnailLink, &iconLink,
			&createdTime, &modifiedTime, &viewedByMeTime, &f.SyncedAt,
		)
		if err != nil {
			return nil, err
		}

		f.MimeType = mimeType.String
		f.FileExtension = fileExt.String
		f.ParentFolderID = parentFolderID.String
		f.ParentFolderName = parentFolderName.String
		f.Path = path.String
		f.SharingUser = sharingUser.String
		f.WebViewLink = webViewLink.String
		f.WebContentLink = webContentLink.String
		f.ThumbnailLink = thumbnailLink.String
		f.IconLink = iconLink.String
		if sizeBytes.Valid {
			f.SizeBytes = sizeBytes.Int64
		}
		if createdTime.Valid {
			f.CreatedTime = createdTime.Time
		}
		if modifiedTime.Valid {
			f.ModifiedTime = modifiedTime.Time
		}
		if viewedByMeTime.Valid {
			f.ViewedByMeTime = viewedByMeTime.Time
		}

		files = append(files, &f)
	}

	return files, nil
}

// GetFile retrieves a single Drive file by file ID.
func (s *DriveService) GetFile(ctx context.Context, userID, fileID string) (*DriveFile, error) {
	var f DriveFile
	var mimeType, fileExt, parentFolderID, parentFolderName, path pgtype.Text
	var sharingUser, webViewLink, webContentLink, thumbnailLink, iconLink pgtype.Text
	var createdTime, modifiedTime, viewedByMeTime pgtype.Timestamptz
	var sizeBytes pgtype.Int8

	err := s.provider.Pool().QueryRow(ctx, `
		SELECT id, user_id, file_id, name, mime_type, file_extension, size_bytes,
			parent_folder_id, parent_folder_name, path, shared, sharing_user,
			web_view_link, web_content_link, thumbnail_link, icon_link,
			created_time, modified_time, viewed_by_me_time, synced_at
		FROM google_drive_files
		WHERE user_id = $1 AND file_id = $2
	`, userID, fileID).Scan(
		&f.ID, &f.UserID, &f.FileID, &f.Name, &mimeType, &fileExt, &sizeBytes,
		&parentFolderID, &parentFolderName, &path, &f.Shared, &sharingUser,
		&webViewLink, &webContentLink, &thumbnailLink, &iconLink,
		&createdTime, &modifiedTime, &viewedByMeTime, &f.SyncedAt,
	)
	if err != nil {
		return nil, err
	}

	f.MimeType = mimeType.String
	f.FileExtension = fileExt.String
	f.ParentFolderID = parentFolderID.String
	f.ParentFolderName = parentFolderName.String
	f.Path = path.String
	f.SharingUser = sharingUser.String
	f.WebViewLink = webViewLink.String
	f.WebContentLink = webContentLink.String
	f.ThumbnailLink = thumbnailLink.String
	f.IconLink = iconLink.String
	if sizeBytes.Valid {
		f.SizeBytes = sizeBytes.Int64
	}
	if createdTime.Valid {
		f.CreatedTime = createdTime.Time
	}
	if modifiedTime.Valid {
		f.ModifiedTime = modifiedTime.Time
	}
	if viewedByMeTime.Valid {
		f.ViewedByMeTime = viewedByMeTime.Time
	}

	return &f, nil
}

// SearchFiles searches Drive files by name.
func (s *DriveService) SearchFiles(ctx context.Context, userID, query string, limit int) ([]*DriveFile, error) {
	rows, err := s.provider.Pool().Query(ctx, `
		SELECT id, user_id, file_id, name, mime_type, file_extension, size_bytes,
			parent_folder_id, shared, web_view_link, modified_time, synced_at
		FROM google_drive_files
		WHERE user_id = $1 AND name ILIKE $2
		ORDER BY modified_time DESC NULLS LAST
		LIMIT $3
	`, userID, "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*DriveFile
	for rows.Next() {
		var f DriveFile
		var mimeType, fileExt, parentFolderID, webViewLink pgtype.Text
		var modifiedTime pgtype.Timestamptz
		var sizeBytes pgtype.Int8

		err := rows.Scan(
			&f.ID, &f.UserID, &f.FileID, &f.Name, &mimeType, &fileExt, &sizeBytes,
			&parentFolderID, &f.Shared, &webViewLink, &modifiedTime, &f.SyncedAt,
		)
		if err != nil {
			return nil, err
		}

		f.MimeType = mimeType.String
		f.FileExtension = fileExt.String
		f.ParentFolderID = parentFolderID.String
		f.WebViewLink = webViewLink.String
		if sizeBytes.Valid {
			f.SizeBytes = sizeBytes.Int64
		}
		if modifiedTime.Valid {
			f.ModifiedTime = modifiedTime.Time
		}

		files = append(files, &f)
	}

	return files, nil
}

// ListFolders lists all folders for a user.
func (s *DriveService) ListFolders(ctx context.Context, userID string) ([]*DriveFile, error) {
	return s.GetFiles(ctx, userID, "application/vnd.google-apps.folder", 1000, 0)
}

// GetRecentFiles gets recently modified files.
func (s *DriveService) GetRecentFiles(ctx context.Context, userID string, limit int) ([]*DriveFile, error) {
	return s.GetFiles(ctx, userID, "", limit, 0)
}

// DeleteFile deletes a file from Google Drive.
func (s *DriveService) DeleteFile(ctx context.Context, userID, fileID string) error {
	srv, err := s.GetDriveAPI(ctx, userID)
	if err != nil {
		return err
	}

	// Delete from Drive
	if err := srv.Files.Delete(fileID).Do(); err != nil {
		return fmt.Errorf("failed to delete from Drive: %w", err)
	}

	// Delete from database
	_, err = s.provider.Pool().Exec(ctx, `
		DELETE FROM google_drive_files WHERE user_id = $1 AND file_id = $2
	`, userID, fileID)

	return err
}

// CreateFolder creates a new folder in Google Drive.
func (s *DriveService) CreateFolder(ctx context.Context, userID, name, parentID string) (*drive.File, error) {
	srv, err := s.GetDriveAPI(ctx, userID)
	if err != nil {
		return nil, err
	}

	folder := &drive.File{
		Name:     name,
		MimeType: "application/vnd.google-apps.folder",
	}

	if parentID != "" {
		folder.Parents = []string{parentID}
	}

	return srv.Files.Create(folder).Fields("id, name, mimeType, webViewLink, createdTime").Do()
}

// UploadFile uploads a file to Google Drive (metadata only - content requires io.Reader).
func (s *DriveService) UploadFile(ctx context.Context, userID, name, mimeType, parentID string) (*drive.File, error) {
	srv, err := s.GetDriveAPI(ctx, userID)
	if err != nil {
		return nil, err
	}

	file := &drive.File{
		Name:     name,
		MimeType: mimeType,
	}

	if parentID != "" {
		file.Parents = []string{parentID}
	}

	return srv.Files.Create(file).Fields("id, name, mimeType, webViewLink, createdTime").Do()
}

// IsConnected checks if Google Drive is connected for a user.
func (s *DriveService) IsConnected(ctx context.Context, userID string) bool {
	// Check if user has drive scopes
	var scopes []string
	err := s.provider.Pool().QueryRow(ctx, `
		SELECT scopes FROM google_oauth_tokens WHERE user_id = $1
	`, userID).Scan(&scopes)
	if err != nil {
		return false
	}

	for _, scope := range scopes {
		if containsDriveScope(scope) {
			return true
		}
	}
	return false
}

func containsDriveScope(scope string) bool {
	driveScopes := []string{
		"https://www.googleapis.com/auth/drive",
		"drive",
		"drive.readonly",
		"drive.file",
	}
	for _, s := range driveScopes {
		if scope == s || scope == "https://www.googleapis.com/auth/"+s {
			return true
		}
	}
	return false
}
