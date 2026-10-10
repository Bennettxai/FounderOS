// Package config loads and validates application configuration from environment
// variables and an optional .env file.
//
// File layout:
//   - config_types.go   — Config struct and AppConfig singleton
//   - config_load.go    — Load(), readDotenvFile(), applyDotenvOverrides()
//   - config_provider.go — AI provider and environment helper methods
//   - config_helpers.go  — Search provider helpers and Validate()
package config
