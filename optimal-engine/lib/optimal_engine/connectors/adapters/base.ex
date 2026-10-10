defmodule OptimalEngine.Connectors.Adapters.Base do
  @moduledoc """
  `use` macro that supplies the boilerplate every adapter needs:

    * declares `@behaviour OptimalEngine.Connectors.Behaviour`
    * imports `Transform` + `Signal` aliases
    * provides `require_keys/2`, `require_credentials/2`, `string_or_atom/2`

  An adapter then writes only the parts that distinguish it:

      defmodule ... do
        use OptimalEngine.Connectors.Adapters.Base, kind: :slack, ...

        @impl true
        def sync(state, cursor), do: ...

        @impl true
        def transform(raw), do: ...
      end
  """

  defmacro __using__(opts) do
    kind = Keyword.fetch!(opts, :kind)
    display_name = Keyword.fetch!(opts, :display_name)
    auth_scheme = Keyword.fetch!(opts, :auth_scheme)
    required_keys = Keyword.get(opts, :required_keys, [])
    credential_keys = Keyword.get(opts, :credential_keys, [])

    quote bind_quoted: [
            kind: kind,
            display_name: display_name,
            auth_scheme: auth_scheme,
            required_keys: required_keys,
            credential_keys: credential_keys
          ] do
      @behaviour OptimalEngine.Connectors.Behaviour

      alias OptimalEngine.Connectors.Transform
      alias OptimalEngine.Signal

      @impl true
      def kind, do: unquote(kind)

      @impl true
      def display_name, do: unquote(display_name)

      @impl true
      def auth_scheme, do: unquote(auth_scheme)

      @impl true
      def required_config_keys, do: unquote(required_keys)

      @impl true
      def init(config) do
        flat = flatten_credentials(config)

        with :ok <- require_keys(flat, unquote(required_keys)),
             :ok <- require_credentials(flat, unquote(credential_keys)) do
          {:ok, hydrate_state(flat)}
        end
      end

      # Adapters override to shape the runtime state they want. Default is
      # to expose the full config map (with credentials lifted to top-level
      # keys).
      def hydrate_state(config) do
        flatten_credentials(config)
      end

      # Adapters with either-or credential schemes (e.g. GitHub pat vs. App,
      # HubSpot token vs. OAuth) declare `credential_keys: []` and override
      # `init/1` to enforce their own credential gate.
      defoverridable init: 1, hydrate_state: 1

      # ── shared helpers ───────────────────────────────────────────────────

      defp require_keys(config, keys) do
        missing =
          Enum.reject(keys, fn k ->
            Map.has_key?(config, k) or Map.has_key?(config, Atom.to_string(k))
          end)

        if missing == [], do: :ok, else: {:error, {:missing_config, missing}}
      end

      defp require_credentials(config, keys) do
        missing =
          Enum.reject(keys, fn k ->
            has_credential_key?(config, k)
          end)

        if missing == [], do: :ok, else: {:error, {:missing_credentials, missing}}
      end

      defp has_credential_key?(config, k) do
        str = if is_atom(k), do: Atom.to_string(k), else: k
        Map.has_key?(config, k) or Map.has_key?(config, str)
      end

      defp pick(config, key, default \\ nil) do
        str = if is_atom(key), do: Atom.to_string(key), else: key
        Map.get(config, key) || Map.get(config, str) || default
      end

      defp flatten_credentials(config) do
        case Map.get(config, "credentials") || Map.get(config, :credentials) do
          creds when is_map(creds) -> Map.merge(config, creds)
          _ -> config
        end
      end
    end
  end
end
