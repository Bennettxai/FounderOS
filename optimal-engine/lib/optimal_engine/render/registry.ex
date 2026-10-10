defmodule OptimalEngine.Render.Registry do
  @moduledoc """
  Loads and serves render specs.

  Specs come from two sources, merged in priority order:

  1. **Workspace specs** — `nodes/<workspace>/.render_specs/*.yaml`
  2. **Built-in defaults** — `priv/render_defaults/*.yaml` (compiled into the release)

  Workspace specs override defaults with the same name. The registry
  is stateless — it reads from disk on every call. For hot-path use,
  wrap in a cache or ETS; for the `/api/render/*` endpoints the
  per-request cost is negligible.
  """

  alias OptimalEngine.Render.Spec

  @defaults_dir Path.join(:code.priv_dir(:optimal_engine) |> to_string(), "render_defaults")

  @doc "List all specs available for a workspace (defaults + workspace overrides)."
  @spec list(String.t()) :: [Spec.t()]
  def list(workspace_id \\ "default") do
    defaults = load_dir(defaults_dir())
    workspace = load_dir(workspace_specs_dir(workspace_id))

    workspace_names = MapSet.new(workspace, & &1.name)

    overridden_defaults =
      Enum.reject(defaults, fn s -> MapSet.member?(workspace_names, s.name) end)

    Enum.sort_by(workspace ++ overridden_defaults, & &1.name)
  end

  @doc "Get a single spec by name, checking workspace first then defaults."
  @spec get(String.t(), String.t()) :: {:ok, Spec.t()} | {:error, :not_found}
  def get(workspace_id \\ "default", name) when is_binary(name) do
    case find_in_dir(workspace_specs_dir(workspace_id), name) do
      {:ok, _} = hit -> hit
      :not_found -> find_in_dir(defaults_dir(), name)
    end
  end

  defp defaults_dir do
    if File.dir?(@defaults_dir) do
      @defaults_dir
    else
      Path.join(File.cwd!(), "lib/optimal_engine/render/defaults")
    end
  end

  defp workspace_specs_dir(workspace_id) do
    Path.join([File.cwd!(), "nodes", workspace_id, ".render_specs"])
  end

  defp load_dir(dir) do
    case File.ls(dir) do
      {:ok, files} ->
        files
        |> Enum.filter(&String.ends_with?(&1, ".yaml"))
        |> Enum.flat_map(fn file ->
          case load_file(Path.join(dir, file)) do
            {:ok, spec} -> [spec]
            _ -> []
          end
        end)

      _ ->
        []
    end
  end

  defp find_in_dir(dir, name) do
    path = Path.join(dir, "#{name}.yaml")

    case load_file(path) do
      {:ok, spec} -> {:ok, spec}
      _ -> :not_found
    end
  end

  defp load_file(path) do
    with {:ok, content} <- File.read(path),
         {:ok, map} <- YamlElixir.read_from_string(content),
         {:ok, spec} <- Spec.from_map(map) do
      {:ok, spec}
    end
  end
end
