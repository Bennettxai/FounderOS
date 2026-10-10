defmodule OptimalEngine.AccessControlStoresTest do
  @moduledoc """
  End-to-end proof that EVERY access-control / identity / multi-user store goes
  from 0 -> >=1 rows via the REAL owning-module flow (not raw SQL hacks), with
  correct shape + provenance.

  Cluster: api_keys, roles, role_grants, groups, principal_groups,
  principal_skills, node_members, workspace_members, sessions.

  Each test counts the store rows for the entities it just created (scoped by
  the unique suffix so it is isolated from any seeded/leftover rows) and proves
  the delta is exactly what the public API produced.
  """
  use ExUnit.Case, async: false

  alias OptimalEngine.Store
  alias OptimalEngine.Identity.{Principal, Group, Role}
  alias OptimalEngine.Auth.ApiKey
  alias OptimalEngine.Topology.{Node, NodeMember, PrincipalSkill, Skill}
  alias OptimalEngine.Session

  @tenant_id "default"

  setup do
    suffix = System.unique_integer([:positive])
    {:ok, suffix: suffix}
  end

  defp count(sql, params) do
    assert {:ok, [[n]]} = Store.raw_query(sql, params)
    n
  end

  # ---- api_keys -----------------------------------------------------------
  test "api_keys: mint goes 0 -> 1 via Auth.ApiKey.mint/1", %{suffix: s} do
    name = "exercise-key-#{s}"
    where = "SELECT COUNT(*) FROM api_keys WHERE name = ?1"
    assert count(where, [name]) == 0

    assert {:ok, %{id: id, key: key}} =
             ApiKey.mint(%{tenant_id: @tenant_id, name: name, scopes: ["read:memory"]})

    assert count(where, [name]) == 1

    # provenance: hashed_secret stored, prefix present, verify round-trips
    assert {:ok, [[hashed, prefix, tid]]} =
             Store.raw_query(
               "SELECT hashed_secret, prefix, tenant_id FROM api_keys WHERE id = ?1",
               [id]
             )

    assert is_binary(hashed) and byte_size(hashed) > 0
    assert is_binary(prefix)
    assert tid == @tenant_id
    assert {:ok, _verified} = ApiKey.verify(key)
  end

  # ---- roles + role_grants + principal -----------------------------------
  test "roles + role_grants: upsert role, grant to principal", %{suffix: s} do
    role_id = "role:sales-#{s}"
    pid = "user:rep-#{s}"

    assert count("SELECT COUNT(*) FROM roles WHERE id = ?1", [role_id]) == 0

    assert {:ok, _} =
             Principal.upsert(%{id: pid, kind: :user, display_name: "Rep #{s}"})

    assert {:ok, _role} =
             Role.upsert(%{id: role_id, name: "sales-#{s}", description: "Sales rep"})

    assert count("SELECT COUNT(*) FROM roles WHERE id = ?1", [role_id]) == 1

    # role_grants 0 -> 1
    assert count("SELECT COUNT(*) FROM role_grants WHERE role_id = ?1", [role_id]) == 0
    assert :ok = Role.grant(%{role_id: role_id, principal_id: pid})
    assert count("SELECT COUNT(*) FROM role_grants WHERE role_id = ?1", [role_id]) == 1

    # provenance: grant is wired to the principal, in the tenant
    assert {:ok, [[gp, gtid]]} =
             Store.raw_query(
               "SELECT principal_id, tenant_id FROM role_grants WHERE role_id = ?1",
               [role_id]
             )

    assert gp == pid
    assert gtid == @tenant_id

    # the read path resolves it (returns a list of role_id strings)
    assert {:ok, roles} = Principal.roles(pid)
    assert role_id in roles
  end

  # ---- groups + principal_groups + group-role grant ----------------------
  test "groups + principal_groups: create group, add member, grant role to group",
       %{suffix: s} do
    gid = "group:legal-#{s}"
    pid = "user:lawyer-#{s}"
    role_id = "role:legalhold-#{s}"

    assert count("SELECT COUNT(*) FROM groups WHERE id = ?1", [gid]) == 0

    assert {:ok, _} =
             Principal.upsert(%{id: pid, kind: :user, display_name: "Lawyer #{s}"})

    assert {:ok, _group} = Group.upsert(%{id: gid, name: "legal-#{s}"})
    assert count("SELECT COUNT(*) FROM groups WHERE id = ?1", [gid]) == 1

    # principal_groups 0 -> 1
    assert count("SELECT COUNT(*) FROM principal_groups WHERE group_id = ?1", [gid]) == 0
    assert :ok = Group.add_member(pid, gid)
    assert count("SELECT COUNT(*) FROM principal_groups WHERE group_id = ?1", [gid]) == 1

    assert {:ok, [[mp]]} =
             Store.raw_query(
               "SELECT principal_id FROM principal_groups WHERE group_id = ?1",
               [gid]
             )

    assert mp == pid

    # group-targeted role grant also exercises role_grants via group_id branch
    assert {:ok, _} = Role.upsert(%{id: role_id, name: "legalhold-#{s}"})
    assert :ok = Role.grant(%{role_id: role_id, group_id: gid})

    assert count("SELECT COUNT(*) FROM role_grants WHERE group_id = ?1", [gid]) == 1
  end

  # ---- principal_skills ---------------------------------------------------
  test "principal_skills: grant a skill to a principal", %{suffix: s} do
    pid = "agent:closer-#{s}"

    assert {:ok, _} =
             Principal.upsert(%{id: pid, kind: :agent, display_name: "Closer #{s}"})

    # skill_id FKs to skills(id) — create the skill via its real flow first
    assert {:ok, %Skill{id: skill_id}} =
             Skill.upsert(%{name: "objection-handling-#{s}", kind: :technical})

    where = "SELECT COUNT(*) FROM principal_skills WHERE principal_id = ?1 AND skill_id = ?2"
    assert count(where, [pid, skill_id]) == 0

    assert :ok =
             PrincipalSkill.grant(pid, skill_id, level: :expert, evidence: "closed 30 deals")

    assert count(where, [pid, skill_id]) == 1

    assert {:ok, [[level, evidence]]} =
             Store.raw_query(
               "SELECT level, evidence FROM principal_skills WHERE principal_id = ?1 AND skill_id = ?2",
               [pid, skill_id]
             )

    assert level == "expert"
    assert evidence == "closed 30 deals"

    assert {:ok, skills} = PrincipalSkill.skills_of(pid)
    assert length(skills) >= 1
  end

  # ---- node_members (2nd principal joins a node) -------------------------
  test "node_members: add a principal to a node", %{suffix: s} do
    {:ok, node} = Node.upsert(%{slug: "ac-node-#{s}", name: "AC Node #{s}", kind: :team})

    pid = "user:member-#{s}"

    assert {:ok, _} =
             Principal.upsert(%{id: pid, kind: :user, display_name: "Member #{s}"})

    where = "SELECT COUNT(*) FROM node_members WHERE node_id = ?1 AND principal_id = ?2"
    assert count(where, [node.id, pid]) == 0

    assert :ok = NodeMember.add(node.id, pid, membership: :internal, role: "lead")
    assert count(where, [node.id, pid]) == 1

    assert {:ok, members} = NodeMember.members_of(node.id)
    assert Enum.any?(members, fn m -> Map.get(m, :principal_id) == pid end)
  end

  # ---- workspace_members (2nd principal joins a workspace) ---------------
  test "workspace_members: add a principal to a workspace", %{suffix: s} do
    slug = "ac-ws-#{s}"

    {:ok, ws} =
      OptimalEngine.Workspace.create(%{slug: slug, name: "AC Workspace #{s}"})

    pid = "user:collab-#{s}"

    assert {:ok, _} =
             Principal.upsert(%{id: pid, kind: :user, display_name: "Collab #{s}"})

    where =
      "SELECT COUNT(*) FROM workspace_members WHERE workspace_id = ?1 AND principal_id = ?2 AND ended_at IS NULL"

    assert count(where, [ws.id, pid]) == 0

    assert {:ok, _} = OptimalEngine.Workspace.add_member(ws.id, pid, role: :owner)
    assert count(where, [ws.id, pid]) == 1

    assert {:ok, [[role]]} =
             Store.raw_query(
               "SELECT role FROM workspace_members WHERE workspace_id = ?1 AND principal_id = ?2",
               [ws.id, pid]
             )

    assert role == "owner"
  end

  # ---- sessions -----------------------------------------------------------
  test "sessions: start + commit persists a row via Session flow" do
    assert {:ok, sid} = Session.start_session(metadata: %{source: "access-control-test"})

    assert count("SELECT COUNT(*) FROM sessions WHERE id = ?1", [sid]) == 0

    :ok = Session.add_message(sid, :user, "hello from the access-control exercise")
    :ok = Session.add_message(sid, :assistant, "acknowledged")

    assert {:ok, _summary} = Session.commit(sid)

    assert count("SELECT COUNT(*) FROM sessions WHERE id = ?1", [sid]) == 1

    assert {:ok, [[mc, summary]]} =
             Store.raw_query(
               "SELECT message_count, summary FROM sessions WHERE id = ?1",
               [sid]
             )

    assert mc == 2
    assert is_binary(summary)
  end
end
