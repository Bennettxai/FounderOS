if Code.ensure_loaded?(:riakc_pb_socket) do
  defmodule OptimalEngine.Knowledge.Backend.Riak do
    @moduledoc """
    Riak KV backend for distributed knowledge storage.

    Uses three Riak buckets mirroring the triple index strategy:
    - `{store_id}_spo` — Subject-Predicate-Object index
    - `{store_id}_pos` — Predicate-Object-Subject index
    - `{store_id}_osp` — Object-Subject-Predicate index

    Each triple is stored as a key in all three buckets. The key encoding
    uses pipe-delimited components for readability and Riak 2i secondary
    indexes for efficient prefix queries.

    ## Requirements

    - A running Riak KV instance (or cluster)
    - `{:riakc, "~> 2.5"}` in deps

    ## Options

    - `:host` — Riak PB host (default: "127.0.0.1")
    - `:port` — Riak PB port (default: 8087)
    - `:bucket_type` — Riak bucket type (default: "default")

    ## Key Design

    Keys: `subject|predicate|object` (SPO), `predicate|object|subject` (POS), etc.
    Values: empty binary (all info is in the key).

    Secondary indexes (2i):
    - `subject_bin` on SPO bucket — enables subject prefix queries
    - `predicate_bin` on POS bucket — enables predicate prefix queries
    - `object_bin` on OSP bucket — enables object prefix queries
    - `sp_bin` on SPO — subject+predicate compound index
    - `po_bin` on POS — predicate+object compound index
    - `os_bin` on OSP — object+subject compound index
    """

    @behaviour OptimalEngine.Knowledge.Backend

    defstruct [:pid, :store_id, :bucket_type, :spo_bucket, :pos_bucket, :osp_bucket]

    @default_host ~c"127.0.0.1"
    @default_port 8087

    @impl true
    def init(store_id, opts) do
      host = Keyword.get(opts, :host, @default_host) |> to_charlist_if_needed()
      port = Keyword.get(opts, :port, @default_port)
      bucket_type = Keyword.get(opts, :bucket_type, "default")

      case :riakc_pb_socket.start_link(host, port) do
        {:ok, pid} ->
          state = %__MODULE__{
            pid: pid,
            store_id: store_id,
            bucket_type: bucket_type,
            spo_bucket: {bucket_type, "#{store_id}_spo"},
            pos_bucket: {bucket_type, "#{store_id}_pos"},
            osp_bucket: {bucket_type, "#{store_id}_osp"}
          }

          {:ok, state}

        {:error, reason} ->
          {:error, {:riak_connection_failed, reason}}
      end
    end

    @impl true
    def assert(state, s, p, o) do
      spo_key = encode_key(s, p, o)
      pos_key = encode_key(p, o, s)
      osp_key = encode_key(o, s, p)

      spo_obj = build_object(state.spo_bucket, spo_key, s, p, :spo)
      pos_obj = build_object(state.pos_bucket, pos_key, p, o, :pos)
      osp_obj = build_object(state.osp_bucket, osp_key, o, s, :osp)

      with :ok <- :riakc_pb_socket.put(state.pid, spo_obj),
           :ok <- :riakc_pb_socket.put(state.pid, pos_obj),
           :ok <- :riakc_pb_socket.put(state.pid, osp_obj) do
        {:ok, state}
      else
        {:error, reason} -> {:error, {:riak_put_failed, reason}}
      end
    end

    @impl true
    def assert_many(state, triples) do
      Enum.reduce_while(triples, {:ok, state}, fn {s, p, o}, {:ok, acc_state} ->
        case assert(acc_state, s, p, o) do
          {:ok, new_state} -> {:cont, {:ok, new_state}}
          error -> {:halt, error}
        end
      end)
    end

    @impl true
    def retract(state, s, p, o) do
      spo_key = encode_key(s, p, o)
      pos_key = encode_key(p, o, s)
      osp_key = encode_key(o, s, p)

      :riakc_pb_socket.delete(state.pid, state.spo_bucket, spo_key)
      :riakc_pb_socket.delete(state.pid, state.pos_bucket, pos_key)
      :riakc_pb_socket.delete(state.pid, state.osp_bucket, osp_key)

      {:ok, state}
    end

    @impl true
    def query(state, pattern) do
      s = Keyword.get(pattern, :subject)
      p = Keyword.get(pattern, :predicate)
      o = Keyword.get(pattern, :object)

      results = do_query(state, s, p, o)
      {:ok, results}
    end

    @impl true
    def count(state) do
      # Use key listing on SPO bucket (expensive but accurate)
      case :riakc_pb_socket.list_keys(state.pid, state.spo_bucket) do
        {:ok, keys} -> {:ok, length(keys)}
        {:error, _} -> {:ok, 0}
      end
    end

    @impl true
    def sparql(_state, _query) do
      {:error, :sparql_not_supported}
    end

    @impl true
    def terminate(state) do
      :riakc_pb_socket.stop(state.pid)
      :ok
    end

    # --- Query implementations using 2i secondary indexes ---

    # All three bound — direct key lookup
    defp do_query(state, s, p, o) when not is_nil(s) and not is_nil(p) and not is_nil(o) do
      key = encode_key(s, p, o)

      case :riakc_pb_socket.get(state.pid, state.spo_bucket, key) do
        {:ok, _obj} -> [{s, p, o}]
        {:error, :notfound} -> []
        _ -> []
      end
    end

    # Subject + Predicate — 2i query on sp_bin
    defp do_query(state, s, p, nil) when not is_nil(s) and not is_nil(p) do
      index_val = "#{s}|#{p}"
      query_2i(state, state.spo_bucket, "sp_bin", index_val, :spo)
    end

    # Subject only — 2i query on subject_bin
    defp do_query(state, s, nil, nil) when not is_nil(s) do
      query_2i(state, state.spo_bucket, "subject_bin", s, :spo)
    end

    # Predicate + Object — 2i query on po_bin
    defp do_query(state, nil, p, o) when not is_nil(p) and not is_nil(o) do
      query_2i(state, state.pos_bucket, "po_bin", "#{p}|#{o}", :pos)
    end

    # Predicate only — 2i query on predicate_bin
    defp do_query(state, nil, p, nil) when not is_nil(p) do
      query_2i(state, state.pos_bucket, "predicate_bin", p, :pos)
    end

    # Object only — 2i query on object_bin
    defp do_query(state, nil, nil, o) when not is_nil(o) do
      query_2i(state, state.osp_bucket, "object_bin", o, :osp)
    end

    # Subject + Object — 2i query on os_bin
    defp do_query(state, s, nil, o) when not is_nil(s) and not is_nil(o) do
      query_2i(state, state.osp_bucket, "os_bin", "#{o}|#{s}", :osp)
    end

    # Wildcard — list all keys from SPO bucket
    defp do_query(state, nil, nil, nil) do
      case :riakc_pb_socket.list_keys(state.pid, state.spo_bucket) do
        {:ok, keys} -> Enum.map(keys, &decode_spo_key/1)
        _ -> []
      end
    end

    # --- 2i helper ---

    defp query_2i(state, bucket, index, value, index_type) do
      case :riakc_pb_socket.get_index_eq(state.pid, bucket, index, value) do
        {:ok, {:index_results_v1, keys, _, _}} ->
          Enum.map(keys, fn key -> decode_key(key, index_type) end)

        _ ->
          []
      end
    end

    # --- Key encoding/decoding ---

    defp encode_key(a, b, c), do: "#{a}|#{b}|#{c}"

    defp decode_spo_key(key), do: decode_key(key, :spo)

    defp decode_key(key, index_type) do
      case String.split(key, "|", parts: 3) do
        [a, b, c] ->
          case index_type do
            :spo -> {a, b, c}
            :pos -> {c, a, b}
            :osp -> {b, c, a}
          end

        _ ->
          {"?", "?", "?"}
      end
    end

    # --- Riak object construction with 2i metadata ---

    defp build_object(bucket, key, idx1, idx2, index_type) do
      obj = :riakc_obj.new(bucket, key, <<>>, "application/octet-stream")

      md = :riakc_obj.get_update_metadata(obj)

      indexes =
        case index_type do
          :spo -> [{"subject_bin", idx1}, {"sp_bin", "#{idx1}|#{idx2}"}]
          :pos -> [{"predicate_bin", idx1}, {"po_bin", "#{idx1}|#{idx2}"}]
          :osp -> [{"object_bin", idx1}, {"os_bin", "#{idx1}|#{idx2}"}]
        end

      md = :riakc_obj.set_secondary_index(md, indexes)
      :riakc_obj.update_metadata(obj, md)
    end

    defp to_charlist_if_needed(val) when is_list(val), do: val
    defp to_charlist_if_needed(val) when is_binary(val), do: String.to_charlist(val)
  end
end

# if Code.ensure_loaded?
