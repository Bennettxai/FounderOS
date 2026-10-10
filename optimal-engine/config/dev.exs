import Config

# Local dev: enable the HTTP API so the desktop can reach the engine.
config :optimal_engine, :api, enabled: true, port: 4200, interface: "127.0.0.1"
