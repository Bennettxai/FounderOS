package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// OneDriveFile represents a OneDrive file or folder.
type OneDriveFile struct {
	ID                      string    `json:"id"`
	UserID                  string    `json:"user_id"`
	ItemID                  string    `json:"item_id"`
	Name                    string    `json:"name"`
	Description             string    `json:"description,omitempty"`
	MimeType                string    `json:"mime_type,omitempty"`
	SizeBytes               int64     `json:"size_bytes,omitempty"`
	ParentReferenceID       string    `json:"parent_reference_id,omitempty"`
	ParentReferencePath     string    `json:"parent_reference_path,omitempty"`
	WebURL                  string    `json:"web_url,omitempty"`
	IsFolder                bool      `json:"is_folder"`
	FolderChildCount        int       `json:"folder_child_count,omitempty"`
	Shared                  bool      `json:"shared"`
	CreatedByUserEmail      string    `json:"created_by_user_email,omitempty"`
	CreatedByUserName       string    `json:"created_by_user_name,omitempty"`
	LastModifiedByUserEmail string    `json:"last_modified_by_user_email,omitempty"`
	LastModifiedByUserName  string    `json:"last_modified_by_user_name,omitempty"`
	CreatedDateTime         time.Time `json:"created_datetime,omitempty"`
	LastModifiedDateTime    time.Time `json:"last_modified_datetime,omitempty"`
	DownloadURL             string    `json:"download_url,omitempty"`
	SyncedAt                time.Time `json:"synced_at"`
}

// OneDriveService handles Microsoft OneDrive operations.
type OneDriveService struct {
	provider *Provider
}

// NewOneDriveService creates a new OneDrive service.
func NewOneDriveService(provider *Provider) *OneDriveService {
	return &OneDriveService{provider: provider}
}

// SyncFiles syncs files from OneDrive.
func (s *OneDriveService) SyncFiles(ctx context.Context, userID string, maxResults int) (*SyncFilesResult, error) {
	slog.Info("OneDrive sync starting for user : max  files", "user_id", userID, "max_results", maxResults)

	client, err := s.provider.GetHTTPClient(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get HTTP client: %w", err)
	}

	result := &SyncFilesResult{}

	// Get root children
	apiURL := fmt.Sprintf("%s/me/drive/root/children?$top=%d&$select=id,name,description,file,folder,size,parentReference,webUrl,shared,createdBy,lastModifiedBy,createdDateTime,lastModifiedDateTime,@microsoft.graph.downloadUrl",
		GraphAPIBase, maxResults)

	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get files: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s", resp.Status)
	}

	var fileResp struct {
		Value []graphDriveItem `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&fileResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	result.TotalFiles = len(fileResp.Value)

	for _, item := range fileResp.Value {
		if err := s.saveFile(ctx, userID, &item); err != nil {
			slog.Info("Failed to save file", "id", item.ID, "error", err)
			result.FailedFiles++
		} else {
			result.SyncedFiles++
		}
	}

	slog.Info("OneDrive sync complete for user : synced / files", "user_id", userID, "synced", result.SyncedFiles, "total", result.TotalFiles)

	return result, nil
}

// SyncFilesResult represents the result of a file sync.
type SyncFilesResult struct {
	TotalFiles  int `json:"total_files"`
	SyncedFiles int `json:"synced_files"`
	FailedFiles int `json:"failed_files"`
}

// Graph API drive item structure
type graphDriveItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Size        int64  `json:"size"`
	WebURL      string `json:"webUrl"`
	File        *struct {
		MimeType string `json:"mimeType"`
	} `json:"file"`
	Folder *struct {
		ChildCount int `json:"childCount"`
	} `json:"folder"`
	ParentReference *struct {
		ID   string `json:"id"`
		Path string `json:"path"`
	} `json:"parentReference"`
	Shared *struct {
		Scope string `json:"scope"`
	} `json:"shared"`
	CreatedBy *struct {
		User struct {
			Email       string `json:"email"`
			DisplayName string `json:"displayName"`
		} `json:"user"`
	} `json:"createdBy"`
	LastModifiedBy *struct {
		User struct {
			Email       string `json:"email"`
			DisplayName string `json:"displayName"`
		} `json:"user"`
	} `json:"lastModifiedBy"`
	CreatedDateTime      string `json:"createdDateTime"`
	LastModifiedDateTime string `json:"lastModifiedDateTime"`
	DownloadURL          string `json:"@microsoft.graph.downloadUrl"`
}

func (s *OneDriveService) saveFile(ctx context.Context, userID string, item *graphDriveItem) error {
	// Determine if folder
	isFolder := item.Folder != nil
	folderChildCount := 0
	if item.Folder != nil {
		folderChildCount = item.Folder.ChildCount
	}

	// Get mime type
	mimeType := ""
	if item.File != nil {
		mimeType = item.File.MimeType
	}

	// Get parent reference
	var parentRefID, parentRefPath string
	if item.ParentReference != nil {
		parentRefID = item.ParentReference.ID
		parentRefPath = item.ParentReference.Path
	}

	// Check if shared
	shared := item.Shared != nil

	// Get created/modified by
	var createdByEmail, createdByName, modifiedByEmail, modifiedByName string
	if item.CreatedBy != nil && item.CreatedBy.User.Email != "" {
		createdByEmail = item.CreatedBy.User.Email
		createdByName = item.CreatedBy.User.DisplayName
	}
	if item.LastModifiedBy != nil && item.LastModifiedBy.User.Email != "" {
		modifiedByEmail = item.LastModifiedBy.User.Email
		modifiedByName = item.LastModifiedBy.User.DisplayName
	}

	// Parse timestamps
	var createdDateTime, lastModifiedDateTime *time.Time
	if item.CreatedDateTime != "" {
		t, _ := time.Parse(time.RFC3339, item.CreatedDateTime)
		createdDateTime = &t
	}
	if item.LastModifiedDateTime != "" {
		t, _ := time.Parse(time.RFC3339, item.LastModifiedDateTime)
		lastModifiedDateTime = &t
	}

	_, err := s.provider.Pool().Exec(ctx, `
		INSERT INTO microsoft_onedrive_files (
			user_id, item_id, name, description, mime_type, size_bytes,
			parent_reference_id, parent_reference_path, web_url,
			is_folder, folder_child_count, shared,
			created_by_user_email, created_by_user_name,
			last_modified_by_user_email, last_modified_by_user_name,
			created_datetime, last_modified_datetime, download_url, synced_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, NOW())
		ON CONFLICT (user_id, item_id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			mime_type = EXCLUDED.mime_type,
			size_bytes = EXCLUDED.size_bytes,
			parent_reference_id = EXCLUDED.parent_reference_id,
			parent_reference_path = EXCLUDED.parent_reference_path,
			web_url = EXCLUDED.web_url,
			is_folder = EXCLUDED.is_folder,
			folder_child_count = EXCLUDED.folder_child_count,
			shared = EXCLUDED.shared,
			last_modified_by_user_email = EXCLUDED.last_modified_by_user_email,
			last_modified_by_user_name = EXCLUDED.last_modified_by_user_name,
			last_modified_datetime = EXCLUDED.last_modified_datetime,
			download_url = EXCLUDED.download_url,
			synced_at = NOW(),
			updated_at = NOW()
	`, userID, item.ID, item.Name, item.Description, mimeType, item.Size,
		parentRefID, parentRefPath, item.WebURL,
		isFolder, folderChildCount, shared,
		createdByEmail, createdByName,
		modifiedByEmail, modifiedByName,
		createdDateTime, lastModifiedDateTime, item.DownloadURL)

	return err
}

// GetFiles retrieves OneDrive files for a user.
func (s *OneDriveService) GetFiles(ctx context.Context, userID string, parentID string, limit, offset int) ([]*OneDriveFile, error) {
	query := `
		SELECT id, user_id, item_id, name, description, mime_type, size_bytes,
			parent_reference_id, parent_reference_path, web_url,
			is_folder, folder_child_count, shared,
			created_by_user_email, created_by_user_name,
			last_modified_by_user_email, last_modified_by_user_name,
			created_datetime, last_modified_datetime, download_url, synced_at
		FROM microsoft_onedrive_files
		WHERE user_id = $1
	`
	args := []interface{}{userID}

	if parentID != "" {
		query += " AND parent_reference_id = $2 ORDER BY name LIMIT $3 OFFSET $4"
		args = append(args, parentID, limit, offset)
	} else {
		query += " ORDER BY last_modified_datetime DESC NULLS LAST LIMIT $2 OFFSET $3"
		args = append(args, limit, offset)
	}

	rows, err := s.provider.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*OneDriveFile
	for rows.Next() {
		var f OneDriveFile
		var description, mimeType, parentRefID, parentRefPath, webURL *string
		var createdByEmail, createdByName, modifiedByEmail, modifiedByName, downloadURL *string
		var createdDateTime, lastModifiedDateTime *time.Time

		err := rows.Scan(
			&f.ID, &f.UserID, &f.ItemID, &f.Name, &description, &mimeType, &f.SizeBytes,
			&parentRefID, &parentRefPath, &webURL,
			&f.IsFolder, &f.FolderChildCount, &f.Shared,
			&createdByEmail, &createdByName,
			&modifiedByEmail, &modifiedByName,
			&createdDateTime, &lastModifiedDateTime, &downloadURL, &f.SyncedAt,
		)
		if err != nil {
			return nil, err
		}

		if description != nil {
			f.Description = *description
		}
		if mimeType != nil {
			f.MimeType = *mimeType
		}
		if parentRefID != nil {
			f.ParentReferenceID = *parentRefID
		}
		if parentRefPath != nil {
			f.ParentReferencePath = *parentRefPath
		}
		if webURL != nil {
			f.WebURL = *webURL
		}
		if createdByEmail != nil {
			f.CreatedByUserEmail = *createdByEmail
		}
		if createdByName != nil {
			f.CreatedByUserName = *createdByName
		}
		if modifiedByEmail != nil {
			f.LastModifiedByUserEmail = *modifiedByEmail
		}
		if modifiedByName != nil {
			f.LastModifiedByUserName = *modifiedByName
		}
		if createdDateTime != nil {
			f.CreatedDateTime = *createdDateTime
		}
		if lastModifiedDateTime != nil {
			f.LastModifiedDateTime = *lastModifiedDateTime
		}
		if downloadURL != nil {
			f.DownloadURL = *downloadURL
		}

		files = append(files, &f)
	}

	return files, nil
}

// SearchFiles searches files by name.
func (s *OneDriveService) SearchFiles(ctx context.Context, userID, query string, limit int) ([]*OneDriveFile, error) {
	rows, err := s.provider.Pool().Query(ctx, `
		SELECT id, user_id, item_id, name, mime_type, size_bytes, web_url, is_folder, synced_at
		FROM microsoft_onedrive_files
		WHERE user_id = $1 AND name ILIKE $2
		ORDER BY last_modified_datetime DESC NULLS LAST
		LIMIT $3
	`, userID, "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*OneDriveFile
	for rows.Next() {
		var f OneDriveFile
		var mimeType, webURL *string

		err := rows.Scan(&f.ID, &f.UserID, &f.ItemID, &f.Name, &mimeType, &f.SizeBytes, &webURL, &f.IsFolder, &f.SyncedAt)
		if err != nil {
			return nil, err
		}

		if mimeType != nil {
			f.MimeType = *mimeType
		}
		if webURL != nil {
			f.WebURL = *webURL
		}

		files = append(files, &f)
	}

	return files, nil
}

// IsConnected checks if OneDrive is connected for a user.
func (s *OneDriveService) IsConnected(ctx context.Context, userID string) bool {
	var scopes []string
	err := s.provider.Pool().QueryRow(ctx, `
		SELECT scopes FROM microsoft_oauth_tokens WHERE user_id = $1
	`, userID).Scan(&scopes)
	if err != nil {
		return false
	}

	for _, scope := range scopes {
		if scope == "Files.Read" || scope == "Files.ReadWrite" || scope == "Files.Read.All" || scope == "Files.ReadWrite.All" {
			return true
		}
	}
	return false
}
