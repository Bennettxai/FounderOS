defmodule OptimalEngine.Embed.Ollama do
  @moduledoc """
  Thin HTTP wrapper around the Ollama local API.

  Stateless module — no GenServer. All calls are synchronous and return
  `{:ok, result}` or `{:error, reason}`.

  Uses Erlang `:httpc` (via `:inets`) so no additional dependencies are required.
  """

  require Logger

  @availability_ttl_ms 60_000
  @availability_cache_key :ollama_availability_cache

  # ---------------------------------------------------------------------------
  # Public API
  # ---------------------------------------------------------------------------

  @doc """
  Returns `true` if the Ollama daemon is reachable, `false` otherwise.

  Result is cached in the calling process dictionary for 60 seconds.
  """
  @spec available?() :: boolean()
  def available? do
    now = System.monotonic_time(:millisecond)

    case Process.get(@availability_cache_key) do
      {result, cached_at} when now - cached_at < @availability_ttl_ms ->
        result

      _ ->
        result = check_availability()
        Process.put(@availability_cache_key, {result, now})
        result
    end
  end

  @doc """
  Stronger than `available?/0` — actually tries an embed probe and
  verifies a non-empty vector comes back. Cached for 60 s per process
  so callers in a tight retrieval loop don't re-probe every call.

  `/api/tags` can be up while `/api/embed` returns empty for missing
  models or partial loads; `available?/0` only catches the former.
  This variant is what the search layer uses to decide whether to
  spend time on a vector hop.
  """
  @spec embed_healthy?() :: boolean()
  def embed_healthy? do
    now = System.monotonic_time(:millisecond)
    key = :oe_ollama_embed_health

    case Process.get(key) do
      {result, cached_at} when now - cached_at < @availability_ttl_ms ->
        result

      _ ->
        # Bound the probe to ~1 s via a dedicated Task so a hung Ollama
        # can't stall the caller for 5–30 s on first use. A healthy
        # local embedder round-trips in tens of milliseconds; anything
        # slower than 1 s is disqualified from the hot retrieval path.
        task =
          Task.async(fn ->
            case embed_text("probe",
                   model:
                     Application.get_env(:optimal_engine, :ollama, [])[:embed_model] ||
                       "nomic-embed-text"
                 ) do
              {:ok, v} when is_list(v) and v != [] -> true
              _ -> false
            end
          end)

        result =
          case Task.yield(task, 1_000) || Task.shutdown(task, :brutal_kill) do
            {:ok, r} when is_boolean(r) -> r
            _ -> false
          end

        Process.put(key, {result, now})
        result
    end
  end

  @doc """
  Generates a text embedding vector for the given input.

  Returns `{:ok, [float()]}` — a 768-dimension vector from nomic-embed-text.

  ## Options
  - `:model` — override the default embed model
  """
  @spec embed(String.t(), keyword()) :: {:ok, [float()]} | {:error, atom()}
  def embed(text, opts \\ []), do: embed_text(text, opts)

  @doc """
  Embed text using the configured text-embedding model.
  Default model: `nomic-embed-text` (configurable via `:optimal_engine, :ollama, :embed_model`).

  Returns `{:ok, [float()]}` — a single 768-dim vector — or `{:error, reason}`.
  """
  @spec embed_text(String.t(), keyword()) :: {:ok, [float()]} | {:error, atom()}
  # Embedding models have a bounded context window; nomic-embed-text rejects
  # oversized input with HTTP 400 ("input length exceeds the context length").
  # Cap input so large chunks embed (truncated) instead of being skipped.
  # nomic-embed-text's effective window is ~2048 tokens; ~4 chars/token, kept
  # conservative so even token-dense text stays under the limit.
  @max_embed_chars 6_000

  def embed_text(text, opts \\ []) when is_binary(text) do
    if String.trim(text) == "" do
      {:error, :empty_input}
    else
      do_embed_text(text, opts)
    end
  end

  defp do_embed_text(text, opts) do
    cfg = config()
    model = Keyword.get(opts, :model, cfg[:embed_model])

    capped =
      if String.length(text) > @max_embed_chars do
        String.slice(text, 0, @max_embed_chars)
      else
        text
      end

    body = %{"model" => model, "input" => capped}

    case post_json("/api/embed", body) do
      {:ok, %{"embeddings" => [first | _]}} when is_list(first) ->
        {:ok, first}

      {:ok, response} ->
        Logger.warning("Ollama embed_text: unexpected response shape: #{inspect(response)}")
        {:error, :unexpected_response}

      {:error, reason} ->
        {:error, reason}
    end
  end

  @doc """
  Embed an image using the configured vision-embedding model.
  Default model: `nomic-embed-vision` — aligned with `nomic-embed-text` in
  the same 768-dim space so text queries can retrieve image chunks.

  Accepts either a filesystem path or raw binary bytes. The image is sent
  base64-encoded in the `images` field of Ollama's `/api/embed` endpoint.

  Returns `{:ok, [float()]}` — a single 768-dim vector — or `{:error, reason}`.
  """
  @spec embed_image(String.t() | binary(), keyword()) :: {:ok, [float()]} | {:error, atom()}
  def embed_image(path_or_bytes, opts \\ [])

  def embed_image(path, opts) when is_binary(path) do
    cond do
      File.exists?(path) ->
        case File.read(path) do
          {:ok, bytes} -> embed_image_bytes(bytes, opts)
          {:error, reason} -> {:error, reason}
        end

      true ->
        embed_image_bytes(path, opts)
    end
  end

  defp embed_image_bytes(bytes, opts) do
    cfg = config()
    model = Keyword.get(opts, :model, cfg[:embed_vision_model] || "nomic-embed-vision")
    encoded = Base.encode64(bytes)

    # Ollama multi-modal embed: image goes in `images` array. `input` carries
    # an optional caption hint — empty string is fine for pure vision embed.
    body = %{"model" => model, "input" => "", "images" => [encoded]}

    case post_json("/api/embed", body) do
      {:ok, %{"embeddings" => [first | _]}} when is_list(first) ->
        {:ok, first}

      {:ok, response} ->
        Logger.warning("Ollama embed_image: unexpected response shape: #{inspect(response)}")
        {:error, :unexpected_response}

      {:error, reason} ->
        {:error, reason}
    end
  end

  @doc """
  Generates a text completion for the given prompt.

  Returns `{:ok, String.t()}` with the model's response.

  ## Options
  - `:model` — override the default generate model
  - `:system` — system prompt string to prepend to the request
  """
  @spec generate(String.t(), keyword()) :: {:ok, String.t()} | {:error, atom()}
  def generate(prompt, opts \\ []) do
    cfg = config()
    model = Keyword.get(opts, :model, cfg[:generate_model])

    body =
      %{"model" => model, "prompt" => prompt, "stream" => false}
      |> maybe_add_system(Keyword.get(opts, :system))

    case post_json("/api/generate", body) do
      {:ok, %{"response" => text}} ->
        {:ok, text}

      {:ok, response} ->
        Logger.warning("Ollama generate: unexpected response shape: #{inspect(response)}")
        {:error, :unexpected_response}

      {:error, reason} ->
        {:error, reason}
    end
  end

  @doc """
  Analyze an image using a Vision-Language Model (VLM).

  Sends the image to a multimodal model (default: `qwen2.5-vl`) via
  Ollama's `/api/generate` endpoint with the `images` array. Returns
  the model's text description of the image.

  Accepts a filesystem path or raw binary bytes. Uses a longer timeout
  than embedding calls since VLM inference is slower.

  ## Options
  - `:model` — override the VLM model (default from config `:vlm_model`)
  - `:system` — system prompt
  - `:timeout_ms` — override the VLM timeout (default from config `:vlm_timeout_ms`)
  """
  @spec vlm_analyze(String.t() | binary(), String.t(), keyword()) ::
          {:ok, String.t()} | {:error, atom()}
  def vlm_analyze(path_or_bytes, prompt, opts \\ [])

  def vlm_analyze(path, prompt, opts) when is_binary(path) and is_binary(prompt) do
    cond do
      File.exists?(path) ->
        case File.read(path) do
          {:ok, bytes} -> vlm_analyze_bytes(bytes, prompt, opts)
          {:error, reason} -> {:error, reason}
        end

      true ->
        vlm_analyze_bytes(path, prompt, opts)
    end
  end

  @doc """
  Returns `true` if the configured VLM model is available in Ollama.

  Checks `/api/tags` for the model name. Cached for 60s per process.
  """
  @spec vlm_available?() :: boolean()
  def vlm_available? do
    now = System.monotonic_time(:millisecond)
    key = :oe_ollama_vlm_available

    case Process.get(key) do
      {result, cached_at} when now - cached_at < @availability_ttl_ms ->
        result

      _ ->
        result = check_vlm_availability()
        Process.put(key, {result, now})
        result
    end
  end

  defp vlm_analyze_bytes(bytes, prompt, opts) do
    cfg = config()
    model = Keyword.get(opts, :model, cfg[:vlm_model] || "qwen2.5-vl")
    timeout = Keyword.get(opts, :timeout_ms, cfg[:vlm_timeout_ms] || 60_000)
    encoded = Base.encode64(bytes)

    body =
      %{"model" => model, "prompt" => prompt, "images" => [encoded], "stream" => false}
      |> maybe_add_system(Keyword.get(opts, :system))

    case post_json("/api/generate", body, timeout) do
      {:ok, %{"response" => text}} ->
        {:ok, text}

      {:ok, response} ->
        Logger.warning("Ollama vlm_analyze: unexpected response shape: #{inspect(response)}")
        {:error, :unexpected_response}

      {:error, reason} ->
        {:error, reason}
    end
  end

  defp check_vlm_availability do
    cfg = config()
    vlm_model = cfg[:vlm_model] || "qwen2.5-vl"

    case get_json("/api/tags") do
      {:ok, %{"models" => models}} when is_list(models) ->
        Enum.any?(models, fn m ->
          name = m["name"] || m["model"] || ""
          String.starts_with?(name, vlm_model)
        end)

      _ ->
        false
    end
  rescue
    _ -> false
  catch
    _, _ -> false
  end

  # ---------------------------------------------------------------------------
  # Private helpers
  # ---------------------------------------------------------------------------

  defp check_availability do
    case get_json("/api/tags") do
      {:ok, _} -> true
      _ -> false
    end
  rescue
    _ -> false
  catch
    _, _ -> false
  end

  defp maybe_add_system(body, nil), do: body
  defp maybe_add_system(body, system), do: Map.put(body, "system", system)

  defp config do
    # 30s was too generous — it let a slow / degraded Ollama stall the
    # Search GenServer long enough that subsequent searches queued
    # behind a dead call. 5s fails fast and lets us degrade to FTS-only
    # retrieval cleanly.
    Application.get_env(:optimal_engine, :ollama,
      host: "http://localhost:11434",
      embed_model: "nomic-embed-text",
      generate_model: "qwen3:8b",
      timeout_ms: 5_000
    )
  end

  defp post_json(path, body, timeout_override \\ nil) do
    :inets.start()

    cfg = config()
    url = (cfg[:host] <> path) |> to_charlist()
    timeout_ms = timeout_override || cfg[:timeout_ms] || 30_000

    headers = [{~c"content-type", ~c"application/json"}]
    content_type = ~c"application/json"
    encoded_body = Jason.encode!(body)

    http_opts = [timeout: timeout_ms, connect_timeout: 5_000]

    case :httpc.request(:post, {url, headers, content_type, encoded_body}, http_opts, []) do
      {:ok, {{_version, status, _reason}, _resp_headers, resp_body}} ->
        parse_response(status, resp_body)

      {:error, reason} ->
        Logger.warning("Ollama HTTP POST #{path} failed: #{inspect(reason)}")
        {:error, :ollama_unavailable}
    end
  end

  defp get_json(path) do
    :inets.start()

    cfg = config()
    url = (cfg[:host] <> path) |> to_charlist()
    timeout_ms = cfg[:timeout_ms] || 30_000

    http_opts = [timeout: timeout_ms, connect_timeout: 5_000]

    case :httpc.request(:get, {url, []}, http_opts, []) do
      {:ok, {{_version, status, _reason}, _resp_headers, resp_body}} ->
        parse_response(status, resp_body)

      {:error, reason} ->
        Logger.warning("Ollama HTTP GET #{path} failed: #{inspect(reason)}")
        {:error, :ollama_unavailable}
    end
  end

  defp parse_response(200, body) do
    case Jason.decode(to_string(body)) do
      {:ok, decoded} ->
        {:ok, decoded}

      {:error, reason} ->
        Logger.warning("Ollama: failed to decode JSON response: #{inspect(reason)}")
        {:error, :invalid_json}
    end
  end

  defp parse_response(status, body) do
    Logger.warning("Ollama: non-200 status #{status}, body: #{to_string(body)}")
    {:error, {:http_error, status}}
  end
end
