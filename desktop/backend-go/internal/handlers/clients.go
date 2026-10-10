package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// ClientHandler handles client management operations
type ClientHandler struct {
	EngineSyncHook
	pool *pgxpool.Pool
}

// NewClientHandler creates a new ClientHandler
func NewClientHandler(pool *pgxpool.Pool) *ClientHandler {
	return &ClientHandler{pool: pool}
}

// ListClients returns all clients for the current user
// clientWorkspaceScope resolves the X-Workspace-ID header and confirms the
// caller is an active member of that workspace (same pattern as the calendar
// module's workspaceScope). Returns (workspaceID, true) when a valid workspace
// context is active, (uuid.Nil, false) otherwise.
func (h *ClientHandler) clientWorkspaceScope(c *gin.Context, userID string) (uuid.UUID, bool) {
	hdr := c.GetHeader("X-Workspace-ID")
	if hdr == "" {
		return uuid.Nil, false
	}
	wsID, err := uuid.Parse(hdr)
	if err != nil {
		return uuid.Nil, false
	}
	var member bool
	err = h.pool.QueryRow(c.Request.Context(),
		`SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND user_id=$2 AND status='active')`,
		wsID, userID).Scan(&member)
	if err != nil || !member {
		return uuid.Nil, false
	}
	return wsID, true
}

// getClientShared fetches a client the caller is allowed to access: their own
// record, OR a workspace-stamped record where the caller is an ACTIVE member of
// that workspace (same trust model as the offers/glossary modules). Records
// with NULL workspace_id remain owner-only. Returns pgx.ErrNoRows when no
// accessible client exists.
func (h *ClientHandler) getClientShared(c *gin.Context, clientID pgtype.UUID, userID string) (sqlc.Client, error) {
	const q = `
		SELECT cl.id, cl.user_id, cl.name, cl.type, cl.email, cl.phone, cl.website,
		       cl.industry, cl.company_size, cl.address, cl.city, cl.state, cl.zip_code,
		       cl.country, cl.status, cl.source, cl.assigned_to, cl.lifetime_value,
		       cl.tags, cl.custom_fields, cl.notes, cl.created_at, cl.updated_at,
		       cl.last_contacted_at
		FROM clients cl
		WHERE cl.id = $1
		  AND (cl.user_id = $2 OR (cl.workspace_id IS NOT NULL AND EXISTS(
		        SELECT 1 FROM workspace_members wm
		        WHERE wm.workspace_id = cl.workspace_id AND wm.user_id = $2 AND wm.status = 'active')))`
	var cl sqlc.Client
	err := h.pool.QueryRow(c.Request.Context(), q, clientID, userID).Scan(
		&cl.ID,
		&cl.UserID,
		&cl.Name,
		&cl.Type,
		&cl.Email,
		&cl.Phone,
		&cl.Website,
		&cl.Industry,
		&cl.CompanySize,
		&cl.Address,
		&cl.City,
		&cl.State,
		&cl.ZipCode,
		&cl.Country,
		&cl.Status,
		&cl.Source,
		&cl.AssignedTo,
		&cl.LifetimeValue,
		&cl.Tags,
		&cl.CustomFields,
		&cl.Notes,
		&cl.CreatedAt,
		&cl.UpdatedAt,
		&cl.LastContactedAt,
	)
	return cl, err
}

func (h *ClientHandler) ListClients(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	queries := sqlc.New(h.pool)

	// Parse optional filters
	var status sqlc.Clientstatus
	if s := c.Query("status"); s != "" {
		status = stringToClientStatus(s)
	}

	var clientType sqlc.Clienttype
	if t := c.Query("type"); t != "" {
		clientType = stringToClientType(t)
	}

	search := c.Query("search")

	pg := ParsePagination(c)

	// Workspace scoping: when an active workspace is selected (X-Workspace-ID,
	// membership verified), show ALL of that workspace's clients regardless of
	// owner, so teammates see the shared book of business. A client belongs to a
	// business context - Agency MIOSA accounts must not leak into other
	// workspaces and vice versa. No header = legacy behavior (all of the user's).
	if wsID, ok := h.clientWorkspaceScope(c, user.ID); ok {
		const wsQuery = `
			SELECT id, user_id, name, type, email, phone, website, industry,
			       company_size, address, city, state, zip_code, country, status,
			       source, assigned_to, lifetime_value, tags, custom_fields, notes,
			       created_at, updated_at, last_contacted_at
			FROM clients
			WHERE workspace_id = $1
			  AND ($2::clientstatus IS NULL OR status = $2)
			  AND ($3::clienttype IS NULL OR type = $3)
			  AND ($4::text IS NULL OR name ILIKE '%' || $4 || '%')
			ORDER BY updated_at DESC`
		rows, err := h.pool.Query(c.Request.Context(), wsQuery,
			wsID,
			sqlc.NullClientstatus{Clientstatus: status, Valid: c.Query("status") != ""},
			sqlc.NullClienttype{Clienttype: clientType, Valid: c.Query("type") != ""},
			utils.StringPtr(search),
		)
		if err != nil {
			utils.RespondInternalError(c, slog.Default(), "list clients", nil)
			return
		}
		defer rows.Close()

		wsClients := []sqlc.Client{}
		for rows.Next() {
			var cl sqlc.Client
			if err := rows.Scan(
				&cl.ID,
				&cl.UserID,
				&cl.Name,
				&cl.Type,
				&cl.Email,
				&cl.Phone,
				&cl.Website,
				&cl.Industry,
				&cl.CompanySize,
				&cl.Address,
				&cl.City,
				&cl.State,
				&cl.ZipCode,
				&cl.Country,
				&cl.Status,
				&cl.Source,
				&cl.AssignedTo,
				&cl.LifetimeValue,
				&cl.Tags,
				&cl.CustomFields,
				&cl.Notes,
				&cl.CreatedAt,
				&cl.UpdatedAt,
				&cl.LastContactedAt,
			); err != nil {
				utils.RespondInternalError(c, slog.Default(), "list clients", nil)
				return
			}
			wsClients = append(wsClients, cl)
		}
		if err := rows.Err(); err != nil {
			utils.RespondInternalError(c, slog.Default(), "list clients", nil)
			return
		}

		all := TransformClients(wsClients)
		total := int64(len(all))
		start := int(pg.Offset)
		end := start + int(pg.Limit)
		if start > len(all) {
			start = len(all)
		}
		if end > len(all) {
			end = len(all)
		}

		c.JSON(http.StatusOK, NewPaginatedResponse(all[start:end], total, pg))
		return
	}

	clients, err := queries.ListClients(c.Request.Context(), sqlc.ListClientsParams{
		UserID:     user.ID,
		Status:     sqlc.NullClientstatus{Clientstatus: status, Valid: c.Query("status") != ""},
		ClientType: sqlc.NullClienttype{Clienttype: clientType, Valid: c.Query("type") != ""},
		Search:     utils.StringPtr(search),
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "list clients", nil)
		return
	}

	// Apply in-memory pagination (SQL query has no LIMIT/OFFSET; all matching rows fetched)
	all := TransformClients(clients)
	total := int64(len(all))
	start := int(pg.Offset)
	end := start + int(pg.Limit)
	if start > len(all) {
		start = len(all)
	}
	if end > len(all) {
		end = len(all)
	}

	c.JSON(http.StatusOK, NewPaginatedResponse(all[start:end], total, pg))
}

// CreateClient creates a new client
func (h *ClientHandler) CreateClient(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req struct {
		Name         string          `json:"name" binding:"required"`
		Type         *string         `json:"type"`
		Email        *string         `json:"email"`
		Phone        *string         `json:"phone"`
		Website      *string         `json:"website"`
		Industry     *string         `json:"industry"`
		CompanySize  *string         `json:"company_size"`
		Address      *string         `json:"address"`
		City         *string         `json:"city"`
		State        *string         `json:"state"`
		ZipCode      *string         `json:"zip_code"`
		Country      *string         `json:"country"`
		Status       *string         `json:"status"`
		Source       *string         `json:"source"`
		AssignedTo   *string         `json:"assigned_to"`
		Tags         json.RawMessage `json:"tags"`
		CustomFields json.RawMessage `json:"custom_fields"`
		Notes        *string         `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	// Parse type
	var clientType sqlc.NullClienttype
	if req.Type != nil {
		clientType = sqlc.NullClienttype{
			Clienttype: stringToClientType(*req.Type),
			Valid:      true,
		}
	}

	// Parse status
	var status sqlc.NullClientstatus
	if req.Status != nil {
		status = sqlc.NullClientstatus{
			Clientstatus: stringToClientStatus(*req.Status),
			Valid:        true,
		}
	}

	// JSON fields: ALWAYS nil in the sqlc insert. Over the Supabase pooler
	// (simple protocol) a non-nil []byte encodes as bytea and jsonb rejects it
	// ("invalid input syntax for type json") - client creates 500'd in prod
	// whenever tags/custom_fields were sent. They are applied post-insert via a
	// text-parameter UPDATE with an explicit ::jsonb cast, which is protocol-safe.
	var tags []byte
	var customFields []byte

	client, err := queries.CreateClient(c.Request.Context(), sqlc.CreateClientParams{
		UserID:       user.ID,
		Name:         req.Name,
		Type:         clientType,
		Email:        req.Email,
		Phone:        req.Phone,
		Website:      req.Website,
		Industry:     req.Industry,
		CompanySize:  req.CompanySize,
		Address:      req.Address,
		City:         req.City,
		State:        req.State,
		ZipCode:      req.ZipCode,
		Country:      req.Country,
		Status:       status,
		Source:       req.Source,
		AssignedTo:   req.AssignedTo,
		Tags:         tags,
		CustomFields: customFields,
		Notes:        req.Notes,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create client", nil)
		return
	}

	// Apply JSON fields post-insert (see comment above: simple-protocol safety).
	if req.Tags != nil || req.CustomFields != nil {
		t := "[]"
		if req.Tags != nil {
			t = string(req.Tags)
		}
		cf := "{}"
		if req.CustomFields != nil {
			cf = string(req.CustomFields)
		}
		if _, err := h.pool.Exec(c.Request.Context(),
			`UPDATE clients SET tags = $1::jsonb, custom_fields = $2::jsonb WHERE id = $3`,
			t, cf, client.ID); err != nil {
			slog.Warn("CreateClient: json fields update failed", "error", err)
		} else {
			client.Tags = []byte(t)
			client.CustomFields = []byte(cf)
		}
	}

	// Stamp the new client with the active workspace (X-Workspace-ID header,
	// membership verified) so it belongs to one business context instead of
	// leaking into every workspace the user opens.
	if wsID, ok := h.clientWorkspaceScope(c, user.ID); ok {
		if _, err := h.pool.Exec(c.Request.Context(),
			`UPDATE clients SET workspace_id = $1 WHERE id = $2`, wsID, client.ID); err != nil {
			slog.Warn("CreateClient: workspace stamp failed", "error", err)
		}
	}

	clientMeta := map[string]string{}
	if req.Type != nil {
		clientMeta["type"] = *req.Type
	}
	if req.Status != nil {
		clientMeta["status"] = *req.Status
	}
	if req.Industry != nil {
		clientMeta["industry"] = *req.Industry
	}
	clientBody := req.Name
	if req.Notes != nil {
		clientBody += "\n\n" + *req.Notes
	}
	h.enqueue(c.Request.Context(), services.Signal{
		Module:     services.ModuleClients,
		ID:         "client-" + uuidString(client.ID.Bytes),
		AuthorID:   user.ID,
		Title:      req.Name,
		Body:       clientBody,
		Genre:      "client",
		ModifiedAt: pgTimestampToTime(client.UpdatedAt),
		Metadata:   clientMeta,
	})

	c.JSON(http.StatusCreated, TransformClient(client))
}

// GetClient returns a single client with its contacts, interactions, and deals
func (h *ClientHandler) GetClient(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "client_id")
		return
	}

	pgID := pgtype.UUID{Bytes: id, Valid: true}
	queries := sqlc.New(h.pool)
	client, err := h.getClientShared(c, pgID, user.ID)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Client")
		return
	}

	// Fetch related data — errors are non-fatal, return empty slices
	contacts, err := queries.ListClientContacts(c.Request.Context(), pgID)
	if err != nil {
		slog.Warn("failed to load client contacts", "client_id", id, "error", err)
		contacts = nil
	}
	interactions, err := queries.ListClientInteractions(c.Request.Context(), pgID)
	if err != nil {
		slog.Warn("failed to load client interactions", "client_id", id, "error", err)
		interactions = nil
	}
	deals, err := queries.ListClientDeals(c.Request.Context(), pgID)
	if err != nil {
		slog.Warn("failed to load client deals", "client_id", id, "error", err)
		deals = nil
	}

	resp := TransformClient(client)
	c.JSON(http.StatusOK, gin.H{
		"id":                resp.ID,
		"user_id":           resp.UserID,
		"name":              resp.Name,
		"type":              resp.Type,
		"email":             resp.Email,
		"phone":             resp.Phone,
		"website":           resp.Website,
		"industry":          resp.Industry,
		"company_size":      resp.CompanySize,
		"address":           resp.Address,
		"city":              resp.City,
		"state":             resp.State,
		"zip_code":          resp.ZipCode,
		"country":           resp.Country,
		"status":            resp.Status,
		"source":            resp.Source,
		"assigned_to":       resp.AssignedTo,
		"lifetime_value":    resp.LifetimeValue,
		"tags":              resp.Tags,
		"custom_fields":     resp.CustomFields,
		"notes":             resp.Notes,
		"created_at":        resp.CreatedAt,
		"updated_at":        resp.UpdatedAt,
		"last_contacted_at": resp.LastContactedAt,
		"contacts":          TransformContacts(contacts),
		"interactions":      TransformInteractions(interactions),
		"deals":             transformClientDealsFromRows(deals),
	})
}

// UpdateClient updates an existing client
func (h *ClientHandler) UpdateClient(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "client_id")
		return
	}

	var req struct {
		Name         *string         `json:"name"`
		Type         *string         `json:"type"`
		Email        *string         `json:"email"`
		Phone        *string         `json:"phone"`
		Website      *string         `json:"website"`
		Industry     *string         `json:"industry"`
		CompanySize  *string         `json:"company_size"`
		Address      *string         `json:"address"`
		City         *string         `json:"city"`
		State        *string         `json:"state"`
		ZipCode      *string         `json:"zip_code"`
		Country      *string         `json:"country"`
		Status       *string         `json:"status"`
		Source       *string         `json:"source"`
		AssignedTo   *string         `json:"assigned_to"`
		Tags         json.RawMessage `json:"tags"`
		CustomFields json.RawMessage `json:"custom_fields"`
		Notes        *string         `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	// Get existing client first (owner or active workspace member)
	existing, err := h.getClientShared(c, pgtype.UUID{Bytes: id, Valid: true}, user.ID)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Client")
		return
	}

	// Build update params with existing values as defaults
	name := existing.Name
	if req.Name != nil {
		name = *req.Name
	}

	clientType := existing.Type
	if req.Type != nil {
		clientType = sqlc.NullClienttype{
			Clienttype: stringToClientType(*req.Type),
			Valid:      true,
		}
	}

	status := existing.Status
	if req.Status != nil {
		status = sqlc.NullClientstatus{
			Clientstatus: stringToClientStatus(*req.Status),
			Valid:        true,
		}
	}

	email := existing.Email
	if req.Email != nil {
		email = req.Email
	}

	phone := existing.Phone
	if req.Phone != nil {
		phone = req.Phone
	}

	website := existing.Website
	if req.Website != nil {
		website = req.Website
	}

	industry := existing.Industry
	if req.Industry != nil {
		industry = req.Industry
	}

	companySize := existing.CompanySize
	if req.CompanySize != nil {
		companySize = req.CompanySize
	}

	address := existing.Address
	if req.Address != nil {
		address = req.Address
	}

	city := existing.City
	if req.City != nil {
		city = req.City
	}

	state := existing.State
	if req.State != nil {
		state = req.State
	}

	zipCode := existing.ZipCode
	if req.ZipCode != nil {
		zipCode = req.ZipCode
	}

	country := existing.Country
	if req.Country != nil {
		country = req.Country
	}

	source := existing.Source
	if req.Source != nil {
		source = req.Source
	}

	assignedTo := existing.AssignedTo
	if req.AssignedTo != nil {
		assignedTo = req.AssignedTo
	}

	tags := existing.Tags
	if req.Tags != nil {
		tags = req.Tags
	}

	customFields := existing.CustomFields
	if req.CustomFields != nil {
		customFields = req.CustomFields
	}

	notes := existing.Notes
	if req.Notes != nil {
		notes = req.Notes
	}

	client, err := queries.UpdateClient(c.Request.Context(), sqlc.UpdateClientParams{
		ID:           pgtype.UUID{Bytes: id, Valid: true},
		Name:         name,
		Type:         clientType,
		Email:        email,
		Phone:        phone,
		Website:      website,
		Industry:     industry,
		CompanySize:  companySize,
		Address:      address,
		City:         city,
		State:        state,
		ZipCode:      zipCode,
		Country:      country,
		Status:       status,
		Source:       source,
		AssignedTo:   assignedTo,
		Tags:         tags,
		CustomFields: customFields,
		Notes:        notes,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update client", nil)
		return
	}

	c.JSON(http.StatusOK, TransformClient(client))
}

// UpdateClientStatus updates only the status of a client
func (h *ClientHandler) UpdateClientStatus(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "client_id")
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	// Verify access (owner or active workspace member)
	_, err = h.getClientShared(c, pgtype.UUID{Bytes: id, Valid: true}, user.ID)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Client")
		return
	}

	client, err := queries.UpdateClientStatus(c.Request.Context(), sqlc.UpdateClientStatusParams{
		ID: pgtype.UUID{Bytes: id, Valid: true},
		Status: sqlc.NullClientstatus{
			Clientstatus: stringToClientStatus(req.Status),
			Valid:        true,
		},
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update client status", nil)
		return
	}

	c.JSON(http.StatusOK, TransformClient(client))
}

// DeleteClient deletes a client
func (h *ClientHandler) DeleteClient(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "client_id")
		return
	}

	queries := sqlc.New(h.pool)
	err = queries.DeleteClient(c.Request.Context(), sqlc.DeleteClientParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "delete client", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Client deleted"})
}

// RegisterClientRoutes registers all client management routes on the given router group.
func RegisterClientRoutes(api *gin.RouterGroup, h *ClientHandler, auth gin.HandlerFunc) {
	clients := api.Group("/clients")
	clients.Use(auth, middleware.RequireAuth())
	{
		clients.GET("", h.ListClients)
		clients.POST("", h.CreateClient)
		clients.GET("/:id", h.GetClient)
		clients.GET("/:id/board", h.GetClientBoard)
		clients.PUT("/:id", h.UpdateClient)
		clients.PATCH("/:id/status", h.UpdateClientStatus)
		clients.DELETE("/:id", h.DeleteClient)
		// Contacts
		clients.GET("/:id/contacts", h.ListClientContacts)
		clients.POST("/:id/contacts", h.CreateClientContact)
		clients.PUT("/:id/contacts/:contactId", h.UpdateClientContact)
		clients.DELETE("/:id/contacts/:contactId", h.DeleteClientContact)
		// Interactions
		clients.GET("/:id/interactions", h.ListClientInteractions)
		clients.POST("/:id/interactions", h.CreateClientInteraction)
		// Deals
		clients.GET("/:id/deals", h.ListClientDeals)
		clients.POST("/:id/deals", h.CreateClientDeal)
		clients.PUT("/:id/deals/:dealId", h.UpdateClientDeal)
	}
}
