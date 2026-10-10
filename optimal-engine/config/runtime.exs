import Config

# BusinessOS runtime config for the BUNDLED OptimalEngine.
#
# This replaces the OptimalOS-private overlay (which hardcoded Roberto's personal
# workspace path). Everything here is env-driven so the desktop EngineManager can
# point the bundled engine at a per-user data directory and a free port, and so a
# downloaded user gets their OWN fresh config from these templated defaults -
# never a developer-machine path.
#
# The EngineManager sets:
#   OPTIMAL_API_ENABLED=true         - turn on the local HTTP API
#   OPTIMAL_API_PORT=<free port>     - avoid colliding with any other engine
#   OPTIMAL_ENGINE_ROOT=<user data>  - where governed workspaces live
#   OPTIMAL_ENGINE_DB / _CACHE       - per-user SQLite + cache (see config.exs)

# Runtime paths must be read here, not only in config.exs. Release builds evaluate
# config.exs at compile time, while downloaded apps supply per-user paths when the
# packaged engine starts.
root_path =
  Application.get_env(
    :optimal_engine,
    :root_path,
    Path.join(File.cwd!(), ".optimal/workspaces")
  )

db_path =
  Application.get_env(:optimal_engine, :db_path, Path.join(File.cwd!(), ".optimal/index.db"))

cache_path =
  Application.get_env(:optimal_engine, :cache_path, Path.join(File.cwd!(), ".optimal/cache"))

topology_path =
  Application.get_env(
    :optimal_engine,
    :topology_path,
    Path.join(File.cwd!(), ".optimal/config.yaml")
  )

topology_full_path =
  Application.get_env(
    :optimal_engine,
    :topology_full_path,
    Path.join(File.cwd!(), ".optimal/topology.yaml")
  )

config :optimal_engine,
  root_path: System.get_env("OPTIMAL_ENGINE_ROOT", root_path),
  db_path: System.get_env("OPTIMAL_ENGINE_DB", db_path),
  cache_path: System.get_env("OPTIMAL_ENGINE_CACHE", cache_path),
  topology_path: System.get_env("OPTIMAL_ENGINE_TOPOLOGY", topology_path),
  topology_full_path: System.get_env("OPTIMAL_ENGINE_TOPOLOGY_FULL", topology_full_path)

knowledge_config = Application.get_env(:optimal_engine, :knowledge, [])
knowledge_backend = Keyword.get(knowledge_config, :backend, "ets") |> to_string()

config :optimal_engine,
       :knowledge,
       Keyword.merge(knowledge_config,
         backend: System.get_env("OPTIMAL_KNOWLEDGE_BACKEND", knowledge_backend),
         rocksdb_path:
           System.get_env(
             "OPTIMAL_KNOWLEDGE_ROCKSDB_PATH",
             Keyword.get(
               knowledge_config,
               :rocksdb_path,
               Path.join(File.cwd!(), ".optimal/knowledge-rocksdb")
             )
           )
       )

ollama_config = Application.get_env(:optimal_engine, :ollama, [])

config :optimal_engine,
       :ollama,
       Keyword.merge(ollama_config,
         host:
           System.get_env(
             "OLLAMA_HOST",
             Keyword.get(ollama_config, :host, "http://localhost:11434")
           ),
         embed_model:
           System.get_env(
             "OPTIMAL_EMBED_MODEL",
             Keyword.get(ollama_config, :embed_model, "nomic-embed-text")
           ),
         generate_model:
           System.get_env(
             "OPTIMAL_GENERATE_MODEL",
             Keyword.get(ollama_config, :generate_model, "qwen3:8b")
           ),
         vlm_model:
           System.get_env(
             "OPTIMAL_VLM_MODEL",
             Keyword.get(ollama_config, :vlm_model, "qwen2.5-vl")
           )
       )

# Local HTTP API - enabled on demand (the bundled engine turns it on; other
# contexts leave it off). Port is env-driven so it can dodge a busy :4200.
if System.get_env("OPTIMAL_API_ENABLED") == "true" do
  config :optimal_engine, :api,
    enabled: true,
    port: String.to_integer(System.get_env("OPTIMAL_API_PORT", "4200")),
    interface: System.get_env("OPTIMAL_API_INTERFACE", "127.0.0.1")
end

# API key enforcement from the environment (prod.exs documents this; the
# The operator bridge staging engines run in dev with auth on).
if System.get_env("OPTIMAL_AUTH_REQUIRED") == "true" do
  config :optimal_engine, :auth, auth_required: true
end
