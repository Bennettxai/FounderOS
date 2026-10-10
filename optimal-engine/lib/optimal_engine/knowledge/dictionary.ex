defmodule OptimalEngine.Knowledge.Dictionary do
  @moduledoc """
  Dictionary encoding for RDF terms — maps string terms to compact 64-bit integer IDs.

  Lock-free dictionary encoding for compact triple storage, implemented in pure
  Elixir using ETS for storage and `:atomics` for a lock-free sequence counter.

  ## ID Layout (64-bit integer)

      ┌──────────┬────────────────────────────────────────────────────────────┐
      │ Bits 63-60│ Bits 59-0                                                │
      │ Type Tag  │ Payload                                                  │
      ├──────────┼────────────────────────────────────────────────────────────┤
      │ 0x1      │ Sequence ID — URI                                        │
      │ 0x2      │ Sequence ID — Blank Node                                 │
      │ 0x3      │ Sequence ID — Literal                                    │
      │ 0x4      │ Inline value — Integer (signed, 60-bit two's complement) │
      └──────────┴────────────────────────────────────────────────────────────┘

  Integers that fit in 60 bits are encoded inline (no ETS lookup needed).
  All other terms go through the bidirectional ETS tables.

  ## Usage

      {:ok, pid} = OptimalEngine.Knowledge.Dictionary.start_link(name: :my_dict)

      {:ok, id} = OptimalEngine.Knowledge.Dictionary.encode(:my_dict, "http://example.org/alice")
      {:ok, "http://example.org/alice"} = OptimalEngine.Knowledge.Dictionary.decode(:my_dict, id)

  ## Concurrency

  The `:atomics` counter provides lock-free ID generation. The GenServer serializes
  only the check-and-insert path to guarantee deduplication. Reads (decode and
  cache-hit encode) go directly to ETS via `:persistent_term`-stored table
  references — zero GenServer involvement on the hot path.
  """

  use GenServer

  import Bitwise

  require Logger

  # -------------------------------------------------------------------
  # Type tags (high 4 bits of a 64-bit ID)
  # -------------------------------------------------------------------
  @tag_uri 0x1
  @tag_bnode 0x2
  @tag_literal 0x3
  @tag_integer 0x4

  @tag_shift 60
  @payload_mask (1 <<< @tag_shift) - 1

  # Maximum magnitude for inline integer encoding (60-bit two's complement).
  @max_inline_int (1 <<< (@tag_shift - 1)) - 1
  @min_inline_int -(1 <<< (@tag_shift - 1))

  # -------------------------------------------------------------------
  # Client API
  # -------------------------------------------------------------------

  @type dict_ref :: GenServer.server()
  @type term_id :: non_neg_integer()

  @doc "Start a dictionary process. Options: `:name` (optional registration name)."
  @spec start_link(keyword()) :: GenServer.on_start()
  def start_link(opts \\ []) do
    {name, init_opts} = Keyword.pop(opts, :name)
    server_opts = if name, do: [name: name], else: []
    GenServer.start_link(__MODULE__, [{:register_as, name} | init_opts], server_opts)
  end

  @doc """
  Encode a string term to its 64-bit integer ID.

  Integers are encoded inline when they fit in 60 bits — pass as
  `{:integer, value}`. All other terms get a monotonic sequence ID
  on first encounter.

  Returns `{:ok, id}` or `{:error, reason}`.
  """
  @spec encode(dict_ref(), String.t() | {:integer, integer()}) ::
          {:ok, term_id()} | {:error, term()}
  def encode(_dict, {:integer, value}) when is_integer(value) do
    if value >= @min_inline_int and value <= @max_inline_int do
      {:ok, encode_inline_integer(value)}
    else
      {:error, {:integer_overflow, value}}
    end
  end

  def encode(dict, term) when is_binary(term) do
    # Fast path: direct ETS read, no GenServer call.
    case ets_lookup_str2id(dict, term) do
      {:ok, _id} = hit -> hit
      :miss -> GenServer.call(dict, {:encode, term})
    end
  end

  @doc """
  Encode multiple terms in a single call. Returns `{:ok, [id]}`.

  Integers in the list should be wrapped as `{:integer, value}`.
  """
  @spec encode_many(dict_ref(), [String.t() | {:integer, integer()}]) :: {:ok, [term_id()]}
  def encode_many(dict, terms) when is_list(terms) do
    {resolved, pending} =
      terms
      |> Enum.with_index()
      |> Enum.reduce({%{}, []}, fn {term, idx}, {res, pend} ->
        case term do
          {:integer, v} when is_integer(v) and v >= @min_inline_int and v <= @max_inline_int ->
            {Map.put(res, idx, encode_inline_integer(v)), pend}

          bin when is_binary(bin) ->
            case ets_lookup_str2id(dict, bin) do
              {:ok, id} -> {Map.put(res, idx, id), pend}
              :miss -> {res, [{idx, bin} | pend]}
            end

          _ ->
            {res, pend}
        end
      end)

    pending_resolved =
      if pending != [] do
        GenServer.call(dict, {:encode_many, Enum.reverse(pending)})
      else
        %{}
      end

    merged = Map.merge(resolved, pending_resolved)
    ids = for i <- 0..(length(terms) - 1), do: Map.fetch!(merged, i)
    {:ok, ids}
  end

  @doc """
  Decode an integer ID back to its string term.

  Inline integers return `{:integer, value}`. All other terms return the
  original string.

  Returns `{:ok, term}` or `{:error, :not_found}`.
  """
  @spec decode(dict_ref(), term_id()) ::
          {:ok, String.t() | {:integer, integer()}} | {:error, :not_found}
  def decode(_dict, id) when is_integer(id) and id >>> @tag_shift == @tag_integer do
    {:ok, {:integer, decode_inline_integer(id)}}
  end

  def decode(dict, id) when is_integer(id) do
    case ets_lookup_id2str(dict, id) do
      {:ok, _} = hit -> hit
      :miss -> {:error, :not_found}
    end
  end

  @doc "Return the type tag atom for a given ID."
  @spec id_type(term_id()) :: :uri | :bnode | :literal | :integer | :unknown
  def id_type(id) when is_integer(id) do
    case id >>> @tag_shift do
      @tag_uri -> :uri
      @tag_bnode -> :bnode
      @tag_literal -> :literal
      @tag_integer -> :integer
      _ -> :unknown
    end
  end

  @doc "Return the current number of allocated (non-inline) term IDs."
  @spec size(dict_ref()) :: non_neg_integer()
  def size(dict) do
    GenServer.call(dict, :size)
  end

  # -------------------------------------------------------------------
  # GenServer Callbacks
  # -------------------------------------------------------------------

  @impl true
  def init(opts) do
    str2id = :ets.new(:dict_str2id, [:set, :protected, read_concurrency: true])
    id2str = :ets.new(:dict_id2str, [:set, :protected, read_concurrency: true])

    counter = :atomics.new(1, signed: false)
    :atomics.put(counter, 1, 1)

    # Publish ETS table refs so client functions can read without GenServer.
    # Use the registered name if available, otherwise the pid.
    register_as = Keyword.get(opts, :register_as)
    pt_key = persistent_term_key(register_as || self())
    :persistent_term.put(pt_key, {str2id, id2str})

    {:ok,
     %{
       str2id: str2id,
       id2str: id2str,
       counter: counter,
       pt_key: pt_key
     }}
  end

  @impl true
  def handle_call({:encode, term}, _from, state) do
    {id, state} = do_encode(term, state)
    {:reply, {:ok, id}, state}
  end

  def handle_call({:encode_many, indexed_terms}, _from, state) do
    results =
      Enum.reduce(indexed_terms, %{}, fn {idx, term}, acc ->
        {id, _state} = do_encode(term, state)
        Map.put(acc, idx, id)
      end)

    {:reply, results, state}
  end

  def handle_call(:size, _from, state) do
    {:reply, :ets.info(state.str2id, :size), state}
  end

  @impl true
  def terminate(_reason, state) do
    :persistent_term.erase(state.pt_key)
    :ets.delete(state.str2id)
    :ets.delete(state.id2str)
    :ok
  end

  # -------------------------------------------------------------------
  # Internal — ID encoding/decoding
  # -------------------------------------------------------------------

  defp do_encode(term, state) do
    case :ets.lookup(state.str2id, term) do
      [{^term, id}] ->
        {id, state}

      [] ->
        tag = classify_term(term)
        seq = :atomics.add_get(state.counter, 1, 1) - 1
        id = tag <<< @tag_shift ||| (seq &&& @payload_mask)

        :ets.insert(state.str2id, {term, id})
        :ets.insert(state.id2str, {id, term})

        {id, state}
    end
  end

  @doc false
  @spec classify_term(String.t()) :: 0x1 | 0x2 | 0x3
  def classify_term("_:" <> _), do: @tag_bnode

  def classify_term(term) do
    if uri?(term), do: @tag_uri, else: @tag_literal
  end

  defp uri?(term) do
    # URI: contains "://" (absolute) or starts with a letter followed by scheme
    # chars then ":" (compact/prefixed URI like "foaf:name").
    String.contains?(term, "://") or
      (String.contains?(term, ":") and
         match?(<<c, _::binary>> when c in ?a..?z or c in ?A..?Z, term))
  end

  defp encode_inline_integer(value) when value >= 0 do
    @tag_integer <<< @tag_shift ||| value
  end

  defp encode_inline_integer(value) when value < 0 do
    @tag_integer <<< @tag_shift ||| (1 <<< @tag_shift) + value
  end

  defp decode_inline_integer(id) do
    raw = id &&& @payload_mask

    if (raw &&& 1 <<< (@tag_shift - 1)) != 0 do
      raw - (1 <<< @tag_shift)
    else
      raw
    end
  end

  # -------------------------------------------------------------------
  # Lock-free ETS reads via persistent_term
  # -------------------------------------------------------------------

  defp ets_lookup_str2id(dict, term) do
    case resolve_tables(dict) do
      {str2id, _id2str} ->
        case :ets.lookup(str2id, term) do
          [{^term, id}] -> {:ok, id}
          [] -> :miss
        end

      :not_found ->
        :miss
    end
  rescue
    ArgumentError -> :miss
  end

  defp ets_lookup_id2str(dict, id) do
    case resolve_tables(dict) do
      {_str2id, id2str} ->
        case :ets.lookup(id2str, id) do
          [{^id, term}] -> {:ok, term}
          [] -> :miss
        end

      :not_found ->
        :miss
    end
  rescue
    ArgumentError -> :miss
  end

  defp resolve_tables(dict) do
    # Try the dict ref directly (atom name or pid).
    key = persistent_term_key(dict)

    try do
      :persistent_term.get(key)
    rescue
      ArgumentError -> :not_found
    end
  end

  defp persistent_term_key(ref), do: {__MODULE__, :tables, ref}
end
