defmodule OptimalEngine.Connectors.Behaviour do
  @moduledoc """
  Contract every enterprise connector must implement.

  A connector is a one-way pipe: "external system X" → "engine intake".
  It has no business logic of its own. The pipeline stages
  (parse / classify / embed / route / store) run downstream of the
  connector's `sync/2` output; a connector's only job is to hand the
  engine a stream of `%Signal{}`s plus an opaque `cursor` that lets
  the next run pick up where this one stopped.

  ## The seven callbacks

      kind/0                 — a stable atom (`:slack`, `:gmail`, …)
      display_name/0         — human label for UIs / logs
      auth_scheme/0          — `:oauth2 | :token | :basic | :webhook`
      required_config_keys/0 — keys that must exist in a connector's
                               `config` JSON before `init/1` runs
      init/1                 — hydrate runtime state (open HTTP client,
                               decode credentials, validate scopes)
      sync/2                 — fetch the next batch of signals, given
                               the cursor returned by the previous run
      transform/1            — map one external payload → `%Signal{}`

  ## The sync contract

  `sync/2` must:

    * return `{:ok, signals, next_cursor}` on success
    * be idempotent — replaying the same cursor yields the same batch
    * bound batch size (the Runner has no memory limit of its own)
    * never raise — all errors are `{:error, reason}` with a reason
      atom the Runner can categorize (`:rate_limited`, `:auth_expired`,
      `:transient`, `:fatal`)

  ## Error categories the Runner honors

      :rate_limited  — wait `retry_after` ms then retry with same cursor
      :auth_expired  — refresh credential, retry once
      :transient     — exponential backoff, up to 5 retries
      :fatal         — mark connector disabled, surface to operator
  """

  alias OptimalEngine.Signal

  @type kind :: atom()
  @type auth :: :oauth2 | :token | :basic | :webhook | :service_account
  @type config :: map()
  @type state :: term()
  @type cursor :: String.t() | nil
  @type raw_payload :: map()
  @type sync_result ::
          {:ok, [Signal.t()], cursor()}
          | {:ok, [Signal.t()], cursor(), [raw_payload()]}
          | {:ok, %{signals: [Signal.t()], cursor: cursor(), payloads: [raw_payload()]}}
  @type sync_error_reason ::
          :rate_limited
          | {:rate_limited, retry_after_ms :: non_neg_integer()}
          | :auth_expired
          | :transient
          | :fatal
          | atom()

  @callback kind() :: kind()
  @callback display_name() :: String.t()
  @callback auth_scheme() :: auth()
  @callback required_config_keys() :: [atom()]
  @callback init(config()) :: {:ok, state()} | {:error, term()}
  @callback sync(state(), cursor()) :: sync_result() | {:error, sync_error_reason()}
  @callback transform(raw_payload()) :: {:ok, Signal.t()} | {:error, term()}

  @optional_callbacks [transform: 1]

  @doc "The full set of valid error-reason atoms."
  @spec known_error_reasons() :: [atom()]
  def known_error_reasons do
    [:rate_limited, :auth_expired, :transient, :fatal]
  end
end
