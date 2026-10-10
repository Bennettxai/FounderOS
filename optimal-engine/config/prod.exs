import Config

# Production overrides — set via environment or runtime config
#
# To require API key authentication in production, set:
#   config :optimal_engine, :auth, auth_required: true, bcrypt_cost: 12
#
# Or (preferred) use runtime.exs to read from the environment:
#   config :optimal_engine, :auth,
#     auth_required: System.get_env("OPTIMAL_AUTH_REQUIRED") == "true",
#     bcrypt_cost: 12
