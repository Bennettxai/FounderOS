import Config

# Use bcrypt cost 4 in tests — functionally identical, ~80x faster than cost 12.
config :optimal_engine, :auth,
  auth_required: false,
  bcrypt_cost: 4

# Integration tests exercise many API endpoints in one run. Keep rate limits high
# so the suite validates behavior rather than hitting infra guardrails.
config :optimal_engine,
  api_rate_limit_capacity: 10_000,
  api_rate_limit_per_minute: 10_000

config :optimal_engine,
  root_path: Path.expand("../tmp/optimal_engine_test_root", System.tmp_dir!()),
  db_path: "/tmp/optimal_engine_test_#{System.get_env("MIX_TEST_PARTITION", "0")}.db",
  cache_path: "/tmp/optimal_engine_test_cache",
  topology_path:
    Path.expand("../tmp/optimal_engine_test_root/.optimal/config.yaml", System.tmp_dir!()),
  topology_full_path:
    Path.expand("../tmp/optimal_engine_test_root/topology.yaml", System.tmp_dir!())

config :optimal_engine, :api_rate_limit,
  default_capacity: 10_000,
  default_per_minute: 10_000

config :optimal_engine, :context_refresh_scheduler,
  enabled: false,
  boot_delay_ms: 1_000,
  interval_ms: 60_000,
  batch_limit: 50,
  workspace_limit: 100

# Unit and integration tests must not depend on a developer's local Ollama.
# Dedicated model integration tests call the adapters explicitly.
config :optimal_engine, :hybrid_search, vector_enabled: false
config :optimal_engine, :embed, on_ingest: false
config :optimal_engine, :semantic_processing, enabled: false
