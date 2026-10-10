package handlers

import "time"

// Computer represents a user's cloud VM instance.
type Computer struct {
	ID          string    `json:"id"`
	Status      string    `json:"status"` // provisioning, active, hibernating, stopped, error
	Plan        string    `json:"plan"`   // free, pro, growth, business
	RAM         int       `json:"ram_gb"` // GB
	CPUs        int       `json:"cpus"`
	StorageGB   int       `json:"storage_gb"`
	StorageUsed float64   `json:"storage_used_gb"`
	Domain      string    `json:"domain"` // e.g. "acme.app.businessos.com"
	Runtimes    []Runtime `json:"runtimes"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Runtime represents a process running inside the cloud computer.
type Runtime struct {
	Name     string `json:"name"`   // "claude-code", "ollama", "custom-agent"
	Status   string `json:"status"` // "active", "idle", "stopped"
	MemoryMB int    `json:"memory_mb"`
	Uptime   int    `json:"uptime_seconds"`
}

// Subscription represents the user's active billing subscription.
type Subscription struct {
	Plan          string    `json:"plan"`          // free, pro, growth, business
	PriceMonthly  int       `json:"price_monthly"` // in cents: 0, 4000, 10000, 20000
	CreditsTotal  int       `json:"credits_total"`
	CreditsUsed   int       `json:"credits_used"`
	SeatsUsed     int       `json:"seats_used"`
	BillingCycle  string    `json:"billing_cycle"` // monthly, yearly
	NextBillingAt time.Time `json:"next_billing_at"`
	Status        string    `json:"status"` // active, past_due, cancelled
}

// PlanInfo describes a purchasable plan tier.
type PlanInfo struct {
	ID           string   `json:"id"` // pro, growth, business
	Name         string   `json:"name"`
	PriceMonthly int      `json:"price_monthly"` // cents
	RAMGB        int      `json:"ram_gb"`
	CPUs         int      `json:"cpus"`
	StorageGB    int      `json:"storage_gb"`
	Credits      int      `json:"credits_included"`
	Features     []string `json:"features"`
}
