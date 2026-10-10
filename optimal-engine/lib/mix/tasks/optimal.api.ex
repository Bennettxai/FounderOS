defmodule Mix.Tasks.Optimal.Api do
  @moduledoc """
  Starts the Optimal Engine HTTP API server.

  The server listens on port 4200 and exposes the backend runtime to apps,
  dashboards, MCP servers, scripts, remote agents, and deployment surfaces.

  ## Usage

      mix optimal.api

  ## Endpoint Groups

      GET  /api/health, /api/status, /api/metrics
      GET  /api/workspaces, /api/workspaces/:id
      GET  /api/search, /api/grep, /api/l0
      POST /api/rag, GET /api/rag/stream
      GET  /api/wiki, /api/wiki/:slug
      POST /api/memory, POST /api/memory/remember
      GET  /api/memory-core/claims
      POST /api/memory-core/claims/:id/promote
      POST /api/memory-core/active-pools
      POST /api/auth/keys
      GET  /api/batch/export/workspace
      GET  /api/graph and related graph analysis routes
  """

  use Mix.Task

  @shortdoc "Start the Optimal Engine HTTP API server on port 4200"

  @impl Mix.Task
  def run(_args) do
    Mix.Task.run("app.start")

    IO.puts("""
    [OptimalEngine API] Listening on http://localhost:4200

    Common endpoint groups:
      GET  /api/health, /api/status, /api/metrics
      GET  /api/workspaces, /api/workspaces/:id
      GET  /api/search?q=<query>, /api/grep?q=<query>, /api/l0
      POST /api/rag
      GET  /api/wiki, /api/wiki/:slug
      POST /api/memory, /api/memory/remember
      GET  /api/memory-core/claims
      POST /api/memory-core/active-pools
      POST /api/auth/keys
      GET  /api/batch/export/workspace
      GET  /api/graph and related graph routes

    Press Ctrl+C to stop.
    """)

    Process.sleep(:infinity)
  end
end
