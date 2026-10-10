package google

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/api/option"
	"google.golang.org/api/people/v1"
)

// Contact represents a synced Google Contact.
type Contact struct {
	ID              string          `json:"id"`
	UserID          string          `json:"user_id"`
	ResourceName    string          `json:"resource_name"` // people/c12345
	DisplayName     string          `json:"display_name,omitempty"`
	GivenName       string          `json:"given_name,omitempty"`
	FamilyName      string          `json:"family_name,omitempty"`
	MiddleName      string          `json:"middle_name,omitempty"`
	Emails          []ContactEmail  `json:"emails,omitempty"`
	PhoneNumbers    []ContactPhone  `json:"phone_numbers,omitempty"`
	Addresses       []ContactAddress `json:"addresses,omitempty"`
	Organization    string          `json:"organization,omitempty"`
	JobTitle        string          `json:"job_title,omitempty"`
	Department      string          `json:"department,omitempty"`
	PhotoURL        string          `json:"photo_url,omitempty"`
	ContactGroups   []string        `json:"contact_groups,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	CreatedTime     time.Time       `json:"created_time,omitempty"`
	ModifiedTime    time.Time       `json:"modified_time,omitempty"`
	SyncedAt        time.Time       `json:"synced_at"`
}

// ContactEmail represents an email address.
type ContactEmail struct {
	Value       string `json:"value"`
	Type        string `json:"type,omitempty"` // home, work, other
	DisplayName string `json:"display_name,omitempty"`
	Primary     bool   `json:"primary,omitempty"`
}

// ContactPhone represents a phone number.
type ContactPhone struct {
	Value string `json:"value"`
	Type  string `json:"type,omitempty"` // mobile, home, work
}

// ContactAddress represents a physical address.
type ContactAddress struct {
	FormattedValue string `json:"formatted_value,omitempty"`
	StreetAddress  string `json:"street_address,omitempty"`
	City           string `json:"city,omitempty"`
	Region         string `json:"region,omitempty"`
	PostalCode     string `json:"postal_code,omitempty"`
	Country        string `json:"country,omitempty"`
	Type           string `json:"type,omitempty"` // home, work
}

// ContactsService handles Google Contacts (People API) operations.
type ContactsService struct {
	provider *Provider
}

// NewContactsService creates a new Contacts service.
func NewContactsService(provider *Provider) *ContactsService {
	return &ContactsService{provider: provider}
}

// GetPeopleAPI returns a Google People API service for a user.
func (s *ContactsService) GetPeopleAPI(ctx context.Context, userID string) (*people.Service, error) {
	tokenSource, err := s.provider.GetTokenSource(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get token source: %w", err)
	}

	srv, err := people.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, fmt.Errorf("failed to create people service: %w", err)
	}

	return srv, nil
}

// SyncContacts syncs contacts from Google Contacts.
func (s *ContactsService) SyncContacts(ctx context.Context, userID string, maxResults int) (*SyncContactsResult, error) {
	slog.Info("Contacts sync starting for user : max  contacts", "user_id", userID, "max_results", maxResults)

	srv, err := s.GetPeopleAPI(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get People API: %w", err)
	}

	// Fields to request (personFields)
	personFields := "names,emailAddresses,phoneNumbers,addresses,organizations,photos,memberships,metadata"

	result := &SyncContactsResult{}
	pageToken := ""

	for {
		req := srv.People.Connections.List("people/me").
			PageSize(int64(maxResults)).
			PersonFields(personFields)

		if pageToken != "" {
			req.PageToken(pageToken)
		}

		resp, err := req.Do()
		if err != nil {
			return nil, fmt.Errorf("failed to list contacts: %w", err)
		}

		result.TotalContacts += len(resp.Connections)

		for _, person := range resp.Connections {
			if err := s.saveContact(ctx, userID, person); err != nil {
				slog.Info("Failed to save contact", "id", person.ResourceName, "error", err)
				result.FailedContacts++
			} else {
				result.SyncedContacts++
			}
		}

		pageToken = resp.NextPageToken
		if pageToken == "" || result.TotalContacts >= maxResults {
			break
		}
	}

	slog.Info("Contacts sync complete for user : synced / contacts", "user_id", userID, "synced", result.SyncedContacts, "total", result.TotalContacts)

	return result, nil
}

// SyncContactsResult represents the result of a contacts sync.
type SyncContactsResult struct {
	TotalContacts  int `json:"total_contacts"`
	SyncedContacts int `json:"synced_contacts"`
	FailedContacts int `json:"failed_contacts"`
}

// saveContact saves a Google Contact to the database.
func (s *ContactsService) saveContact(ctx context.Context, userID string, person *people.Person) error {
	// Extract display name
	var displayName, givenName, familyName, middleName string
	if len(person.Names) > 0 {
		name := person.Names[0]
		displayName = name.DisplayName
		givenName = name.GivenName
		familyName = name.FamilyName
		middleName = name.MiddleName
	}

	// Extract emails
	emails := make([]ContactEmail, 0)
	for _, e := range person.EmailAddresses {
		emails = append(emails, ContactEmail{
			Value:       e.Value,
			Type:        e.Type,
			DisplayName: e.DisplayName,
		})
	}

	// Extract phone numbers
	phones := make([]ContactPhone, 0)
	for _, p := range person.PhoneNumbers {
		phones = append(phones, ContactPhone{
			Value: p.Value,
			Type:  p.Type,
		})
	}

	// Extract addresses
	addresses := make([]ContactAddress, 0)
	for _, a := range person.Addresses {
		addresses = append(addresses, ContactAddress{
			FormattedValue: a.FormattedValue,
			StreetAddress:  a.StreetAddress,
			City:           a.City,
			Region:         a.Region,
			PostalCode:     a.PostalCode,
			Country:        a.Country,
			Type:           a.Type,
		})
	}

	// Extract organization
	var organization, jobTitle, department string
	if len(person.Organizations) > 0 {
		org := person.Organizations[0]
		organization = org.Name
		jobTitle = org.Title
		department = org.Department
	}

	// Extract photo
	var photoURL string
	if len(person.Photos) > 0 {
		photoURL = person.Photos[0].Url
	}

	// Extract contact groups
	groups := make([]string, 0)
	for _, m := range person.Memberships {
		if m.ContactGroupMembership != nil {
			groups = append(groups, m.ContactGroupMembership.ContactGroupId)
		}
	}

	// Extract metadata times
	var createdTime, modifiedTime *time.Time
	if person.Metadata != nil {
		for _, source := range person.Metadata.Sources {
			if source.UpdateTime != "" {
				t, _ := time.Parse(time.RFC3339, source.UpdateTime)
				modifiedTime = &t
			}
		}
	}

	// Insert or update contact
	_, err := s.provider.Pool().Exec(ctx, `
		INSERT INTO google_contacts (
			user_id, resource_name, display_name, given_name, family_name, middle_name,
			emails, phone_numbers, addresses,
			organization, job_title, department, photo_url,
			contact_groups, created_time, modified_time, synced_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, NOW())
		ON CONFLICT (user_id, resource_name) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			given_name = EXCLUDED.given_name,
			family_name = EXCLUDED.family_name,
			middle_name = EXCLUDED.middle_name,
			emails = EXCLUDED.emails,
			phone_numbers = EXCLUDED.phone_numbers,
			addresses = EXCLUDED.addresses,
			organization = EXCLUDED.organization,
			job_title = EXCLUDED.job_title,
			department = EXCLUDED.department,
			photo_url = EXCLUDED.photo_url,
			contact_groups = EXCLUDED.contact_groups,
			modified_time = EXCLUDED.modified_time,
			synced_at = NOW(),
			updated_at = NOW()
	`, userID, person.ResourceName, displayName, givenName, familyName, middleName,
		emails, phones, addresses,
		organization, jobTitle, department, photoURL,
		groups, createdTime, modifiedTime)

	return err
}

// GetContacts retrieves contacts for a user.
func (s *ContactsService) GetContacts(ctx context.Context, userID string, limit, offset int) ([]*Contact, error) {
	rows, err := s.provider.Pool().Query(ctx, `
		SELECT id, user_id, resource_name, display_name, given_name, family_name, middle_name,
			emails, phone_numbers, addresses,
			organization, job_title, department, photo_url,
			contact_groups, created_time, modified_time, synced_at
		FROM google_contacts
		WHERE user_id = $1
		ORDER BY display_name
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []*Contact
	for rows.Next() {
		var c Contact
		var displayName, givenName, familyName, middleName pgtype.Text
		var organization, jobTitle, department, photoURL pgtype.Text
		var emails, phones, addresses, groups []byte
		var createdTime, modifiedTime pgtype.Timestamptz

		err := rows.Scan(
			&c.ID, &c.UserID, &c.ResourceName, &displayName, &givenName, &familyName, &middleName,
			&emails, &phones, &addresses,
			&organization, &jobTitle, &department, &photoURL,
			&groups, &createdTime, &modifiedTime, &c.SyncedAt,
		)
		if err != nil {
			return nil, err
		}

		c.DisplayName = displayName.String
		c.GivenName = givenName.String
		c.FamilyName = familyName.String
		c.MiddleName = middleName.String
		c.Organization = organization.String
		c.JobTitle = jobTitle.String
		c.Department = department.String
		c.PhotoURL = photoURL.String
		if createdTime.Valid {
			c.CreatedTime = createdTime.Time
		}
		if modifiedTime.Valid {
			c.ModifiedTime = modifiedTime.Time
		}

		contacts = append(contacts, &c)
	}

	return contacts, nil
}

// GetContact retrieves a single contact by resource name.
func (s *ContactsService) GetContact(ctx context.Context, userID, resourceName string) (*Contact, error) {
	var c Contact
	var displayName, givenName, familyName, middleName pgtype.Text
	var organization, jobTitle, department, photoURL pgtype.Text
	var emails, phones, addresses, groups []byte
	var createdTime, modifiedTime pgtype.Timestamptz

	err := s.provider.Pool().QueryRow(ctx, `
		SELECT id, user_id, resource_name, display_name, given_name, family_name, middle_name,
			emails, phone_numbers, addresses,
			organization, job_title, department, photo_url,
			contact_groups, created_time, modified_time, synced_at
		FROM google_contacts
		WHERE user_id = $1 AND resource_name = $2
	`, userID, resourceName).Scan(
		&c.ID, &c.UserID, &c.ResourceName, &displayName, &givenName, &familyName, &middleName,
		&emails, &phones, &addresses,
		&organization, &jobTitle, &department, &photoURL,
		&groups, &createdTime, &modifiedTime, &c.SyncedAt,
	)
	if err != nil {
		return nil, err
	}

	c.DisplayName = displayName.String
	c.GivenName = givenName.String
	c.FamilyName = familyName.String
	c.MiddleName = middleName.String
	c.Organization = organization.String
	c.JobTitle = jobTitle.String
	c.Department = department.String
	c.PhotoURL = photoURL.String
	if createdTime.Valid {
		c.CreatedTime = createdTime.Time
	}
	if modifiedTime.Valid {
		c.ModifiedTime = modifiedTime.Time
	}

	return &c, nil
}

// SearchContacts searches contacts by name or email.
func (s *ContactsService) SearchContacts(ctx context.Context, userID, query string, limit int) ([]*Contact, error) {
	rows, err := s.provider.Pool().Query(ctx, `
		SELECT id, user_id, resource_name, display_name, given_name, family_name,
			organization, job_title, photo_url, synced_at
		FROM google_contacts
		WHERE user_id = $1
			AND (display_name ILIKE $2 OR given_name ILIKE $2 OR family_name ILIKE $2 OR organization ILIKE $2)
		ORDER BY display_name
		LIMIT $3
	`, userID, "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []*Contact
	for rows.Next() {
		var c Contact
		var displayName, givenName, familyName, organization, jobTitle, photoURL pgtype.Text

		err := rows.Scan(
			&c.ID, &c.UserID, &c.ResourceName, &displayName, &givenName, &familyName,
			&organization, &jobTitle, &photoURL, &c.SyncedAt,
		)
		if err != nil {
			return nil, err
		}

		c.DisplayName = displayName.String
		c.GivenName = givenName.String
		c.FamilyName = familyName.String
		c.Organization = organization.String
		c.JobTitle = jobTitle.String
		c.PhotoURL = photoURL.String

		contacts = append(contacts, &c)
	}

	return contacts, nil
}

// CreateContact creates a new contact in Google Contacts.
func (s *ContactsService) CreateContact(ctx context.Context, userID string, contact *Contact) (*people.Person, error) {
	srv, err := s.GetPeopleAPI(ctx, userID)
	if err != nil {
		return nil, err
	}

	person := &people.Person{
		Names: []*people.Name{
			{
				GivenName:  contact.GivenName,
				FamilyName: contact.FamilyName,
				MiddleName: contact.MiddleName,
			},
		},
	}

	// Add emails
	for _, e := range contact.Emails {
		person.EmailAddresses = append(person.EmailAddresses, &people.EmailAddress{
			Value: e.Value,
			Type:  e.Type,
		})
	}

	// Add phone numbers
	for _, p := range contact.PhoneNumbers {
		person.PhoneNumbers = append(person.PhoneNumbers, &people.PhoneNumber{
			Value: p.Value,
			Type:  p.Type,
		})
	}

	// Add organization
	if contact.Organization != "" || contact.JobTitle != "" {
		person.Organizations = []*people.Organization{
			{
				Name:       contact.Organization,
				Title:      contact.JobTitle,
				Department: contact.Department,
			},
		}
	}

	return srv.People.CreateContact(person).Do()
}

// DeleteContact deletes a contact from Google Contacts.
func (s *ContactsService) DeleteContact(ctx context.Context, userID, resourceName string) error {
	srv, err := s.GetPeopleAPI(ctx, userID)
	if err != nil {
		return err
	}

	// Delete from Google
	_, err = srv.People.DeleteContact(resourceName).Do()
	if err != nil {
		return fmt.Errorf("failed to delete contact from Google: %w", err)
	}

	// Delete from database
	_, err = s.provider.Pool().Exec(ctx, `
		DELETE FROM google_contacts WHERE user_id = $1 AND resource_name = $2
	`, userID, resourceName)

	return err
}

// IsConnected checks if Google Contacts is connected for a user.
func (s *ContactsService) IsConnected(ctx context.Context, userID string) bool {
	var scopes []string
	err := s.provider.Pool().QueryRow(ctx, `
		SELECT scopes FROM google_oauth_tokens WHERE user_id = $1
	`, userID).Scan(&scopes)
	if err != nil {
		return false
	}

	for _, scope := range scopes {
		if containsContactsScope(scope) {
			return true
		}
	}
	return false
}

func containsContactsScope(scope string) bool {
	contactScopes := []string{
		"https://www.googleapis.com/auth/contacts",
		"contacts",
		"contacts.readonly",
	}
	for _, s := range contactScopes {
		if scope == s || scope == "https://www.googleapis.com/auth/"+s {
			return true
		}
	}
	return false
}
