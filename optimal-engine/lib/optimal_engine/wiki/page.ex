defmodule OptimalEngine.Wiki.Page do
  @moduledoc """
  A single wiki page — the atomic unit of Tier 3.

  Mirrors the `wiki_pages` table (Phase 1 migration 008):

      %Page{tenant_id, slug, audience, version, frontmatter, body,
            last_curated, curated_by}

  The `body` carries the rendered-markdown view with **unresolved** directive
  syntax (`{{cite: optimal://...}}`, `{{include: ...}}`, etc.) — rendering
  into plain text / Claude XML / OpenAI messages happens at READ time via
  `OptimalEngine.Wiki.Directives.render/3`.

  Pages are keyed on `(tenant_id, slug, audience, version)`. Audience is
  the mechanism by which the same source signal can produce per-role views
  (`sales` / `engineering` / `exec-brief` / `default`) — different curator
  runs, different citations passed through the intersection filter, same
  slug.
  """

  @type audience :: String.t()

  @type t :: %__MODULE__{
          tenant_id: String.t(),
          workspace_id: String.t(),
          slug: String.t(),
          audience: audience(),
          version: non_neg_integer(),
          frontmatter: map(),
          body: String.t(),
          last_curated: String.t() | nil,
          curated_by: String.t() | nil
        }

  defstruct tenant_id: "default",
            workspace_id: "default",
            slug: nil,
            audience: "default",
            version: 1,
            frontmatter: %{},
            body: "",
            last_curated: nil,
            curated_by: nil

  @frontmatter_re ~r/\A---\r?\n(.*?)\r?\n---\r?\n?(.*)\z/s

  @doc """
  Parse a markdown document (optionally with YAML-ish frontmatter) into a
  Page struct. Frontmatter is parsed permissively — plain `key: value`
  pairs; more structured YAML is supported via `yaml_elixir` if the
  frontmatter contains a `yaml` marker.

  Options:
    * `:tenant_id` — default: `"default"`
    * `:slug`      — default: derived from frontmatter `slug` or filename
    * `:audience`  — default: `"default"`
  """
  @spec from_markdown(String.t(), keyword()) :: {:ok, t()} | {:error, term()}
  def from_markdown(markdown, opts \\ []) when is_binary(markdown) do
    {frontmatter, body} = parse_frontmatter(markdown)

    page = %__MODULE__{
      tenant_id: Keyword.get(opts, :tenant_id, Map.get(frontmatter, "tenant_id", "default")),
      workspace_id:
        Keyword.get(opts, :workspace_id, Map.get(frontmatter, "workspace_id", "default")),
      slug: Keyword.get(opts, :slug, Map.get(frontmatter, "slug")),
      audience: Keyword.get(opts, :audience, Map.get(frontmatter, "audience", "default")),
      version: Map.get(frontmatter, "version", 1) |> to_integer_safe(1),
      frontmatter: frontmatter,
      body: body,
      last_curated: Map.get(frontmatter, "last_curated"),
      curated_by: Map.get(frontmatter, "curated_by")
    }

    if is_nil(page.slug) do
      {:error, :missing_slug}
    else
      {:ok, page}
    end
  end

  @doc "Serialize a Page back to markdown (frontmatter block + body)."
  @spec to_markdown(t()) :: String.t()
  def to_markdown(%__MODULE__{} = page) do
    fm =
      page.frontmatter
      |> Map.put("slug", page.slug)
      |> Map.put("audience", page.audience)
      |> Map.put("version", page.version)
      |> Map.put("tenant_id", page.tenant_id)
      |> Map.put_new_lazy("last_curated", fn -> page.last_curated end)
      |> Map.put_new_lazy("curated_by", fn -> page.curated_by end)

    yaml = render_frontmatter(fm)
    "---\n" <> yaml <> "---\n\n" <> page.body
  end

  @doc "Build a fresh Page with reasonable defaults."
  @spec new(keyword()) :: t()
  def new(fields) when is_list(fields) do
    struct(__MODULE__, fields)
  end

  # ─── private ─────────────────────────────────────────────────────────────

  defp parse_frontmatter(markdown) do
    case Regex.run(@frontmatter_re, markdown) do
      [_, yaml_block, body] ->
        {parse_yaml_block(yaml_block), String.trim_leading(body, "\n")}

      _ ->
        {%{}, markdown}
    end
  end

  # Permissive "YAML-ish" parser: handles `key: value` pairs + nested lists
  # via `yaml_elixir` when the content starts with something complex. For
  # simple key-value frontmatter we use our own parser so we don't have to
  # require yaml_elixir's full machinery.
  defp parse_yaml_block(yaml) do
    case YamlElixir.read_from_string(yaml) do
      {:ok, parsed} when is_map(parsed) -> parsed
      {:ok, _} -> fallback_parse(yaml)
      {:error, _} -> fallback_parse(yaml)
    end
  end

  defp fallback_parse(yaml) do
    yaml
    |> String.split("\n", trim: true)
    |> Enum.reduce(%{}, fn line, acc ->
      case String.split(line, ":", parts: 2) do
        [k, v] -> Map.put(acc, String.trim(k), String.trim(v))
        _ -> acc
      end
    end)
  end

  defp render_frontmatter(map) do
    map
    |> Enum.reject(fn {_k, v} -> is_nil(v) end)
    |> Enum.sort_by(fn {k, _} -> k end)
    |> Enum.map_join("\n", fn {k, v} -> "#{k}: #{format_yaml_value(v)}" end)
    |> Kernel.<>("\n")
  end

  defp format_yaml_value(v) when is_binary(v), do: v
  defp format_yaml_value(v) when is_number(v) or is_atom(v), do: to_string(v)

  defp format_yaml_value(list) when is_list(list) do
    "[" <> Enum.map_join(list, ", ", &format_yaml_value/1) <> "]"
  end

  defp format_yaml_value(other), do: inspect(other)

  defp to_integer_safe(v, _fallback) when is_integer(v), do: v

  defp to_integer_safe(v, fallback) when is_binary(v) do
    case Integer.parse(v) do
      {n, _} -> n
      _ -> fallback
    end
  end

  defp to_integer_safe(_, fallback), do: fallback
end
