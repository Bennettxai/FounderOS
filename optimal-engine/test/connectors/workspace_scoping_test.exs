defmodule OptimalEngine.Connectors.WorkspaceScopingTest do
  @moduledoc """
  T3 workspace isolation: connectors carry a workspace_id so every signal a
  connector ingests can be attributed to the connector's workspace.

  Covers:
    * registration persists workspace_id (migration 035 column)
    * run bookkeeping (connector_runs) carries the connector's workspace
    * the signal sink receives the connector's workspace scope, so a
      connector registered in workspace B lands signals scoped to B
  """

  use ExUnit.Case, async: false

  alias OptimalEngine.Connectors
  alias OptimalEngine.Connectors.Runner
  alias OptimalEngine.Store

  defmodule StubAdapter do
    @moduledoc "Behaviour-compliant adapter that returns one synthetic signal."
    @behaviour OptimalEngine.Connectors.Behaviour

    @impl true
    def kind, do: :slack

    @impl true
    def display_name, do: "Stub"

    @impl true
    def auth_scheme, do: :token

    @impl true
    def required_config_keys, do: []

    @impl true
    def init(config), do: {:ok, config}

    @impl true
    def sync(_state, _cursor) do
      signal = %OptimalEngine.Signal{
        id: "stub-signal",
        title: "Stub signal",
        content: "stub content",
        node: "inbox",
        sn_ratio: 0.5,
        entities: [],
        l0_summary: "stub",
        l1_description: "stub",
        routed_to: []
      }

      {:ok, [signal], "cursor-next"}
    end
  end

  setup do
    suffix = System.unique_integer([:positive])

    {:ok, id: "ws-scope-connector-#{suffix}", workspace: "connector-ws-b-#{suffix}"}
  end

  test "register persists the connector's workspace_id", %{id: id, workspace: ws} do
    {:ok, ^id} =
      Connectors.register(%{
        id: id,
        kind: :slack,
        workspace_id: ws,
        config: %{
          "workspace_id" => "T01",
          "channels" => ["C01"],
          "credentials" => %{"bot_token" => "xoxb-test"}
        }
      })

    {:ok, [[stored_ws]]} =
      Store.raw_query("SELECT workspace_id FROM connectors WHERE id = ?1", [id])

    assert stored_ws == ws
  end

  test "register without workspace_id defaults to 'default'", %{id: id} do
    {:ok, ^id} =
      Connectors.register(%{id: id, kind: :slack, config: %{}})

    {:ok, [[stored_ws]]} =
      Store.raw_query("SELECT workspace_id FROM connectors WHERE id = ?1", [id])

    assert stored_ws == "default"
  end

  test "connector_runs rows carry the connector's workspace_id", %{id: id, workspace: ws} do
    {:ok, ^id} =
      Connectors.register(%{
        id: id,
        kind: :slack,
        workspace_id: ws,
        config: %{
          "workspace_id" => "T01",
          "channels" => ["C01"],
          "credentials" => %{"bot_token" => "xoxb-test"}
        }
      })

    # Slack's real sync is :not_implemented — the run errors, but the run row
    # must still be attributed to the connector's workspace.
    {:ok, result} = Runner.run(id)
    assert result.workspace_id == ws

    {:ok, [[run_ws]]} =
      Store.raw_query(
        "SELECT workspace_id FROM connector_runs WHERE connector_id = ?1 ORDER BY id DESC LIMIT 1",
        [id]
      )

    assert run_ws == ws
  end

  test "signal sink receives the connector's workspace scope on success", %{
    id: id,
    workspace: ws
  } do
    {:ok, ^id} =
      Connectors.register(%{id: id, kind: :slack, workspace_id: ws, config: %{}})

    parent = self()

    sink = fn signals, scope ->
      send(parent, {:sink_called, signals, scope})
      :ok
    end

    {:ok, result} = Runner.run(id, adapter: StubAdapter, signal_sink: sink)

    assert result.status == :success
    assert result.workspace_id == ws
    assert result.signals == 1

    assert_receive {:sink_called, [signal], scope}
    assert signal.id == "stub-signal"
    assert scope.workspace_id == ws
    assert scope.connector_id == id
  end

  test "legacy arity-1 sinks still work", %{id: id, workspace: ws} do
    {:ok, ^id} =
      Connectors.register(%{id: id, kind: :slack, workspace_id: ws, config: %{}})

    parent = self()

    sink = fn signals ->
      send(parent, {:legacy_sink, length(signals)})
      :ok
    end

    {:ok, result} = Runner.run(id, adapter: StubAdapter, signal_sink: sink)
    assert result.status == :success
    assert_receive {:legacy_sink, 1}
  end
end
