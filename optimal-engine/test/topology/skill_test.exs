defmodule OptimalEngine.Topology.SkillTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Identity.Principal
  alias OptimalEngine.Topology.{PrincipalSkill, Skill}

  describe "Skill.upsert/1" do
    test "creates a skill with defaults" do
      name = "skill-#{System.unique_integer([:positive])}"

      assert {:ok, %Skill{name: ^name, kind: :technical}} =
               Skill.upsert(%{name: name, kind: :technical})

      assert {:ok, listed} = Skill.list(kind: :technical)
      assert Enum.any?(listed, &(&1.name == name))
    end

    test "rejects invalid kind" do
      assert {:error, {:invalid_kind, :weird}} =
               Skill.upsert(%{name: "z-#{System.unique_integer([:positive])}", kind: :weird})
    end

    test "is idempotent" do
      name = "idem-#{System.unique_integer([:positive])}"
      assert {:ok, s1} = Skill.upsert(%{name: name, description: "v1"})
      assert {:ok, s2} = Skill.upsert(%{name: name, description: "v2"})

      assert s1.id == s2.id
      assert {:ok, fetched} = Skill.get(s1.id)
      assert fetched.description == "v2"
    end
  end

  describe "PrincipalSkill grants" do
    setup do
      suffix = System.unique_integer([:positive])

      {:ok, user} =
        Principal.upsert(%{
          id: "user:skill-test-#{suffix}",
          kind: :user,
          display_name: "Skill Test User #{suffix}"
        })

      {:ok, agent} =
        Principal.upsert(%{
          id: "agent:skill-test-#{suffix}",
          kind: :agent,
          display_name: "Skill Test Agent #{suffix}"
        })

      {:ok, skill} = Skill.upsert(%{name: "test-capability-#{suffix}", kind: :technical})

      {:ok, user: user, agent: agent, skill: skill}
    end

    test "grant + skills_of round-trips",
         %{user: %{id: user_id}, skill: %{id: skill_id}} do
      assert :ok = PrincipalSkill.grant(user_id, skill_id, level: :expert, evidence: "5y")
      assert {:ok, skills} = PrincipalSkill.skills_of(user_id)
      assert Enum.any?(skills, &(&1.skill_id == skill_id and &1.level == :expert))
    end

    test "min_level filter drops below threshold",
         %{user: %{id: user_id}, skill: %{id: skill_id}} do
      :ok = PrincipalSkill.grant(user_id, skill_id, level: :novice)

      assert {:ok, all} = PrincipalSkill.skills_of(user_id)
      assert Enum.any?(all, &(&1.skill_id == skill_id))

      assert {:ok, filtered} = PrincipalSkill.skills_of(user_id, min_level: :expert)
      refute Enum.any?(filtered, &(&1.skill_id == skill_id))
    end

    test "principals_with_skill returns both users and agents",
         %{user: %{id: user_id}, agent: %{id: agent_id}, skill: %{id: skill_id}} do
      :ok = PrincipalSkill.grant(user_id, skill_id, level: :expert)
      :ok = PrincipalSkill.grant(agent_id, skill_id, level: :intermediate)

      assert {:ok, holders} = PrincipalSkill.principals_with_skill(skill_id)
      holder_ids = Enum.map(holders, & &1.principal_id)
      assert user_id in holder_ids
      assert agent_id in holder_ids

      kinds = Enum.map(holders, & &1.principal_kind) |> Enum.uniq() |> Enum.sort()
      assert :user in kinds
      assert :agent in kinds
    end

    test "re-grant updates level + evidence",
         %{user: %{id: user_id}, skill: %{id: skill_id}} do
      :ok = PrincipalSkill.grant(user_id, skill_id, level: :novice, evidence: "just starting")
      :ok = PrincipalSkill.grant(user_id, skill_id, level: :lead, evidence: "runs the team")

      {:ok, skills} = PrincipalSkill.skills_of(user_id)
      grant = Enum.find(skills, &(&1.skill_id == skill_id))
      assert grant.level == :lead
      assert grant.evidence == "runs the team"
    end

    test "revoke removes the grant",
         %{user: %{id: user_id}, skill: %{id: skill_id}} do
      :ok = PrincipalSkill.grant(user_id, skill_id)
      :ok = PrincipalSkill.revoke(user_id, skill_id)
      {:ok, skills} = PrincipalSkill.skills_of(user_id)
      refute Enum.any?(skills, &(&1.skill_id == skill_id))
    end
  end
end
