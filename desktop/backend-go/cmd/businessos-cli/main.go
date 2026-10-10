// businessos is a CLI tool for managing BusinessOS modules inside a Computer (Firecracker VM).
// It talks to the local Go backend at http://localhost:8001.
//
// Usage:
//
//	businessos module create --name "Inventory" --category "custom" --icon "package"
//	businessos module install <directory>
//	businessos module list
//	businessos module remove <slug>
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"
)

const (
	defaultBackendURL = "http://localhost:8001"
	apiBase           = "/api/v1/modules"
)

// manifest is the on-disk format stored in manifest.json inside a module directory.
type manifest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Icon        string `json:"icon"`
	Version     string `json:"version"`
	Route       string `json:"route"`
	Author      string `json:"author"`
}

// createModuleRequest mirrors the backend's CreateModuleRequest.
type createModuleRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Manifest    map[string]interface{} `json:"manifest"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Icon        string                 `json:"icon,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Keywords    []string               `json:"keywords,omitempty"`
}

// moduleResponse is the minimal shape we need from the backend list/create response.
type moduleResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Category    string  `json:"category"`
	Version     string  `json:"version"`
	Description *string `json:"description"`
	Icon        *string `json:"icon"`
	IsPublished bool    `json:"is_published"`
}

// listResponse matches the paginated envelope from ListModules.
type listResponse struct {
	Data  []moduleResponse `json:"data"`
	Total int64            `json:"total"`
}

// httpClient is the shared HTTP client with a sensible timeout.
var httpClient = &http.Client{Timeout: 15 * time.Second}

func backendURL() string {
	if u := os.Getenv("BUSINESSOS_API_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	return defaultBackendURL
}

// --- entry point ---------------------------------------------------------------

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "module":
		handleModule(os.Args[2:])
	case "--help", "-h", "help":
		printUsage()
	default:
		fatalf("unknown command %q\n\nRun 'businessos help' for usage.", os.Args[1])
	}
}

// --- command dispatcher --------------------------------------------------------

func handleModule(args []string) {
	if len(args) == 0 {
		printModuleUsage()
		os.Exit(1)
	}

	switch args[0] {
	case "create":
		cmdModuleCreate(args[1:])
	case "install":
		cmdModuleInstall(args[1:])
	case "list":
		cmdModuleList()
	case "remove":
		cmdModuleRemove(args[1:])
	default:
		fatalf("unknown module subcommand %q\n\nRun 'businessos module' for usage.", args[0])
	}
}

// --- module create -------------------------------------------------------------

func cmdModuleCreate(args []string) {
	var (
		name     string
		category string
		icon     string
	)

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--name":
			name = nextArg(args, &i, "--name")
		case "--category":
			category = nextArg(args, &i, "--category")
		case "--icon":
			icon = nextArg(args, &i, "--icon")
		default:
			fatalf("unknown flag %q for 'module create'", args[i])
		}
	}

	if name == "" {
		fatalf("--name is required\n\nUsage: businessos module create --name <name> [--category <category>] [--icon <icon>]")
	}
	if category == "" {
		category = "custom"
	}
	if icon == "" {
		icon = "box"
	}

	slug := toSlug(name)
	dir := slug

	// Refuse to overwrite an existing directory.
	if _, err := os.Stat(dir); err == nil {
		fatalf("directory %q already exists", dir)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		fatalf("failed to create directory %q: %v", dir, err)
	}

	mf := manifest{
		Name:        name,
		Slug:        slug,
		Description: fmt.Sprintf("Track and manage %s", strings.ToLower(name)),
		Category:    category,
		Icon:        icon,
		Version:     "1.0.0",
		Route:       "/" + slug,
		Author:      "user",
	}
	writeJSON(filepath.Join(dir, "manifest.json"), mf)
	writeFile(filepath.Join(dir, "page.svelte"), pageSvelteTemplate(name, slug))
	writeFile(filepath.Join(dir, "handler.go"), handlerGoTemplate(name))
	writeFile(filepath.Join(dir, "migration.sql"), migrationSQLTemplate(name))
	writeFile(filepath.Join(dir, "README.md"), readmeTemplate(name, slug))

	fmt.Printf("Module scaffold created: %s/\n", dir)
	fmt.Printf("  manifest.json\n")
	fmt.Printf("  page.svelte\n")
	fmt.Printf("  handler.go\n")
	fmt.Printf("  migration.sql\n")
	fmt.Printf("  README.md\n")
	fmt.Printf("\nNext: businessos module install ./%s\n", dir)
}

// --- module install ------------------------------------------------------------

func cmdModuleInstall(args []string) {
	if len(args) == 0 {
		fatalf("directory argument required\n\nUsage: businessos module install <directory>")
	}
	dir := args[0]

	// Read manifest.
	mf, err := readManifest(filepath.Join(dir, "manifest.json"))
	if err != nil {
		fatalf("failed to read manifest.json: %v", err)
	}

	// Build the manifest map to embed in the request.
	manifestMap := map[string]interface{}{
		"name":    mf.Name,
		"slug":    mf.Slug,
		"version": mf.Version,
		"route":   mf.Route,
		"author":  mf.Author,
		"icon":    mf.Icon,
	}

	req := createModuleRequest{
		Name:        mf.Name,
		Description: mf.Description,
		Category:    mf.Category,
		Icon:        mf.Icon,
		Tags:        []string{},
		Keywords:    []string{},
		Manifest:    manifestMap,
		Config:      map[string]interface{}{},
	}

	body, err := json.Marshal(req)
	if err != nil {
		fatalf("failed to encode request: %v", err)
	}

	resp, err := apiPost(apiBase, body)
	if err != nil {
		fatalf("API error: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		fatalf("backend returned %d: %s", resp.StatusCode, string(respBody))
	}

	var mod moduleResponse
	if err := json.Unmarshal(respBody, &mod); err != nil {
		// Non-fatal: print raw if parse fails.
		fmt.Printf("Module installed: %s (raw response: %s)\n", mf.Name, string(respBody))
		return
	}

	fmt.Printf("Module installed: %s (id: %s)\n", mod.Name, mod.ID)
}

// --- module list ---------------------------------------------------------------

func cmdModuleList() {
	resp, err := apiGet(apiBase)
	if err != nil {
		fatalf("API error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		fatalf("backend returned %d: %s", resp.StatusCode, string(body))
	}

	modules, err := parseModuleList(body)
	if err != nil {
		fatalf("failed to parse response: %v", err)
	}

	if len(modules) == 0 {
		fmt.Println("No modules found.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tCATEGORY\tVERSION\tSTATUS")
	for _, m := range modules {
		status := "inactive"
		if m.IsPublished {
			status = "active"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", m.Name, m.Category, m.Version, status)
	}
	w.Flush()
}

// --- module remove -------------------------------------------------------------

func cmdModuleRemove(args []string) {
	if len(args) == 0 {
		fatalf("slug argument required\n\nUsage: businessos module remove <slug>")
	}
	slug := args[0]

	// Resolve slug → id via list.
	id, err := resolveModuleID(slug)
	if err != nil {
		fatalf("failed to resolve module %q: %v", slug, err)
	}

	resp, err := apiDelete(apiBase + "/" + id)
	if err != nil {
		fatalf("API error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		fatalf("backend returned %d: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("Module removed: %s\n", slug)
}

// --- API helpers ---------------------------------------------------------------

func apiGet(path string) (*http.Response, error) {
	return httpClient.Get(backendURL() + path)
}

func apiPost(path string, body []byte) (*http.Response, error) {
	return httpClient.Post(backendURL()+path, "application/json", bytes.NewReader(body))
}

func apiDelete(path string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodDelete, backendURL()+path, nil)
	if err != nil {
		return nil, err
	}
	return httpClient.Do(req)
}

// resolveModuleID lists all modules and returns the ID for the given slug.
func resolveModuleID(slug string) (string, error) {
	resp, err := apiGet(apiBase)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("backend returned %d: %s", resp.StatusCode, string(body))
	}

	modules, err := parseModuleList(body)
	if err != nil {
		return "", err
	}

	for _, m := range modules {
		if m.Slug == slug || m.ID == slug {
			return m.ID, nil
		}
	}
	return "", errors.New("module not found")
}

// parseModuleList handles both a paginated envelope {data:[...]} and a bare array [...].
func parseModuleList(body []byte) ([]moduleResponse, error) {
	var list listResponse
	if err := json.Unmarshal(body, &list); err == nil && list.Data != nil {
		return list.Data, nil
	}
	var modules []moduleResponse
	if err := json.Unmarshal(body, &modules); err != nil {
		return nil, fmt.Errorf("failed to parse module list: %v", err)
	}
	return modules, nil
}

// --- file helpers -------------------------------------------------------------

func readManifest(path string) (manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return manifest{}, err
	}
	var mf manifest
	if err := json.Unmarshal(data, &mf); err != nil {
		return manifest{}, fmt.Errorf("invalid JSON: %w", err)
	}
	if mf.Name == "" {
		return manifest{}, errors.New("manifest missing required field: name")
	}
	return mf, nil
}

func writeJSON(path string, v interface{}) {
	data, err := json.MarshalIndent(v, "", "    ")
	if err != nil {
		fatalf("failed to marshal %s: %v", path, err)
	}
	writeFile(path, string(data))
}

func writeFile(path, content string) {
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		fatalf("failed to write %s: %v", path, err)
	}
}

// --- slug / string helpers ----------------------------------------------------

// toSlug converts a display name to a lowercase-hyphenated slug.
// "My Inventory" → "my-inventory"
func toSlug(name string) string {
	s := strings.ToLower(name)
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, s)
	// Collapse multiple hyphens and trim edges.
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.Trim(s, "-")
}

func nextArg(args []string, i *int, flag string) string {
	*i++
	if *i >= len(args) {
		fatalf("flag %s requires a value", flag)
	}
	return args[*i]
}

func fatalf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", a...)
	os.Exit(1)
}

// --- usage text ---------------------------------------------------------------

func printUsage() {
	fmt.Print(`businessos — BusinessOS module management

USAGE
    businessos <command> [arguments]

COMMANDS
    module    Manage custom modules

Run 'businessos module' for module subcommands.
`)
}

func printModuleUsage() {
	fmt.Print(`businessos module — manage custom modules

USAGE
    businessos module <subcommand> [arguments]

SUBCOMMANDS
    create    Scaffold a new module directory
    install   Register a module directory with the backend
    list      List all modules
    remove    Delete a module by slug

EXAMPLES
    businessos module create --name "Inventory" --category "custom" --icon "package"
    businessos module install ./inventory
    businessos module list
    businessos module remove inventory

FLAGS (create)
    --name      Module display name (required)
    --category  Module category (default: custom)
    --icon      Icon name (default: box)

ENVIRONMENT
    BUSINESSOS_API_URL   Backend URL (default: http://localhost:8001)
`)
}

// --- scaffold templates -------------------------------------------------------

func pageSvelteTemplate(name, slug string) string {
	return fmt.Sprintf(`<script>
  let items = $state([]);

  async function loadData() {
    const res = await fetch('/api/v1/modules/%s/data');
    items = await res.json();
  }
</script>

<div class="module-page">
  <h1>%s</h1>
  <!-- Module UI here -->
</div>

<style>
  .module-page {
    padding: 1.5rem;
  }
</style>
`, slug, name)
}

func handlerGoTemplate(name string) string {
	return fmt.Sprintf(`package main

// %s module API handler
// Register routes in the BusinessOS backend by implementing http.Handler.
`, name)
}

func migrationSQLTemplate(name string) string {
	return fmt.Sprintf(`-- %s module database schema
-- Add your tables here

-- Example:
-- CREATE TABLE IF NOT EXISTS %s_items (
--     id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
--     workspace_id UUID NOT NULL,
--     name        TEXT NOT NULL,
--     created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
--     updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
-- );
`, name, toSlug(name))
}

func readmeTemplate(name, slug string) string {
	return fmt.Sprintf(`# %s Module

A custom BusinessOS module.

## Install

    businessos module install .

## Files

| File | Purpose |
|------|---------|
| manifest.json | Module metadata |
| page.svelte | Frontend UI |
| handler.go | Backend API handler |
| migration.sql | Database schema |

## Route

This module mounts at **/%s**.
`, name, slug)
}
