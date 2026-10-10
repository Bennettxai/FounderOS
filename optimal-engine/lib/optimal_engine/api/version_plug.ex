defmodule OptimalEngine.API.VersionPlug do
  @moduledoc """
  Rewrites `/v1/...` requests to `/api/...` so the existing router handles them.
  `/api/...` continues to work unchanged (backward compatibility).

  Clients should prefer `/v1/` going forward; `/api/` is a permanent compat alias.
  The rewrite is transparent — no redirect overhead, just a path mutation before
  Plug.Router.match/2 runs.
  """

  @behaviour Plug

  @impl Plug
  def init(opts), do: opts

  @impl Plug
  def call(%{request_path: "/v1/" <> rest} = conn, _opts) do
    %{conn | request_path: "/api/" <> rest, path_info: ["api" | tl(conn.path_info)]}
  end

  def call(conn, _opts), do: conn
end
