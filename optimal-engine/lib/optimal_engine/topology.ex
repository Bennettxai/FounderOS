defmodule OptimalEngine.Topology do
  @moduledoc """
  Top-level facade for the organizational topology layer.

  An organization's workspace is the first-class model of the operation it represents:
  nodes (organizational units), people (principal.kind=:user), agents
  (principal.kind=:agent), skills (capabilities), memberships (who's in
  what node), and tool integrations (the translation layer between the
  human and agent layers).

  This module is a thin facade over:

    * `OptimalEngine.Topology.Node`             — organizational units
    * `OptimalEngine.Topology.NodeMember`       — principal ↔ node memberships
    * `OptimalEngine.Topology.Skill`            — capability registry
    * `OptimalEngine.Topology.PrincipalSkill`   — principal ↔ skill grants

  See `docs/architecture/WORKSPACE.md` for the model.
  """

  alias OptimalEngine.Organization
  alias OptimalEngine.Topology.{Node, NodeMember, NodeRelationship, NodeType, PrincipalSkill, Skill}

  # ── Organizations ────────────────────────────────────────────────────────

  defdelegate create_organization(attrs), to: Organization, as: :create
  defdelegate get_organization(id), to: Organization, as: :get
  defdelegate list_organizations(opts), to: Organization, as: :list

  # ── Nodes ────────────────────────────────────────────────────────────────

  defdelegate create_node(attrs), to: Node, as: :upsert
  defdelegate get_node(id, tenant_id), to: Node, as: :get
  defdelegate get_node_by_slug(slug, tenant_id), to: Node, as: :get_by_slug
  defdelegate list_nodes(opts), to: Node, as: :list
  defdelegate children(node_id, opts), to: Node
  defdelegate ancestors(node_id, opts), to: Node
  defdelegate node_tree(opts), to: Node, as: :tree

  # ── Node Types ───────────────────────────────────────────────────────────

  defdelegate create_node_type(attrs), to: NodeType, as: :upsert
  defdelegate get_node_type(id, opts), to: NodeType, as: :get
  defdelegate get_node_type_by_slug(slug, opts), to: NodeType, as: :get_by_slug
  defdelegate list_node_types(opts), to: NodeType, as: :list

  # ── Node Relationships ──────────────────────────────────────────────────

  defdelegate create_node_relationship(attrs), to: NodeRelationship, as: :upsert
  defdelegate get_node_relationship(id, opts), to: NodeRelationship, as: :get
  defdelegate relationships_for_node(node_id, opts), to: NodeRelationship, as: :for_node

  # ── Memberships ──────────────────────────────────────────────────────────

  defdelegate add_member(node_id, principal_id, opts), to: NodeMember, as: :add
  defdelegate remove_member(node_id, principal_id, opts), to: NodeMember, as: :remove
  defdelegate members_of(node_id, opts), to: NodeMember
  defdelegate nodes_of(principal_id, opts), to: NodeMember

  # ── Skills ───────────────────────────────────────────────────────────────

  defdelegate create_skill(attrs), to: Skill, as: :upsert
  defdelegate get_skill(id, tenant_id), to: Skill, as: :get
  defdelegate list_skills(opts), to: Skill, as: :list

  defdelegate grant_skill(principal_id, skill_id, opts), to: PrincipalSkill, as: :grant
  defdelegate revoke_skill(principal_id, skill_id, opts), to: PrincipalSkill, as: :revoke
  defdelegate skills_of(principal_id, opts), to: PrincipalSkill
  defdelegate principals_with_skill(skill_id, opts), to: PrincipalSkill
end
