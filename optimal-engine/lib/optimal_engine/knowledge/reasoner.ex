defmodule OptimalEngine.Knowledge.Reasoner do
  @moduledoc """
  OWL 2 RL forward-chaining reasoner with semi-naive evaluation.

  Materializes inferred triples by applying RDFS and OWL 2 RL entailment rules
  repeatedly until a fixed-point is reached (no new triples produced).

  Backend-agnostic: works with any module implementing `OptimalEngine.Knowledge.Backend`.
  """

  require Logger

  # --- RDF/RDFS/OWL URIs ---

  @rdf_type "http://www.w3.org/1999/02/22-rdf-syntax-ns#type"
  @rdfs_subclass "http://www.w3.org/2000/01/rdf-schema#subClassOf"
  @rdfs_subproperty "http://www.w3.org/2000/01/rdf-schema#subPropertyOf"
  @rdfs_domain "http://www.w3.org/2000/01/rdf-schema#domain"
  @rdfs_range "http://www.w3.org/2000/01/rdf-schema#range"
  @owl_same_as "http://www.w3.org/2002/07/owl#sameAs"
  @owl_inverse_of "http://www.w3.org/2002/07/owl#inverseOf"
  @owl_transitive "http://www.w3.org/2002/07/owl#TransitiveProperty"
  @owl_symmetric "http://www.w3.org/2002/07/owl#SymmetricProperty"
  @owl_equivalent_class "http://www.w3.org/2002/07/owl#equivalentClass"
  @owl_equivalent_property "http://www.w3.org/2002/07/owl#equivalentProperty"
  @owl_has_value "http://www.w3.org/2002/07/owl#hasValue"
  @owl_on_property "http://www.w3.org/2002/07/owl#onProperty"

  @default_max_rounds 100

  # --- Public API ---

  @doc """
  Run full materialization on a store. Applies rules in rounds until fixed-point.

  Returns `{:ok, new_state, rounds}` where `rounds` is the number of rounds executed.
  """
  @spec materialize(module(), term(), keyword()) :: {:ok, term(), non_neg_integer()}
  def materialize(backend, state, opts \\ []) do
    max_rounds = Keyword.get(opts, :max_rounds, @default_max_rounds)
    materialize_loop(backend, state, 0, max_rounds)
  end

  @doc """
  Run a single round of all rule applications.

  Returns `{:ok, new_state, new_triples_count}`.
  """
  @spec apply_rules(module(), term()) :: {:ok, term(), non_neg_integer()}
  def apply_rules(backend, state) do
    # Collect all existing triples into a set for fast dedup
    {:ok, existing} = backend.query(state, [])
    existing_set = MapSet.new(existing)

    # Fire all rules and collect inferred triples
    inferred =
      rules()
      |> Enum.flat_map(fn rule_fn -> rule_fn.(backend, state) end)
      |> MapSet.new()
      |> MapSet.difference(existing_set)

    # Assert new triples
    new_triples = MapSet.to_list(inferred)
    count = length(new_triples)

    if count > 0 do
      {:ok, new_state} = backend.assert_many(state, new_triples)
      {:ok, new_state, count}
    else
      {:ok, state, 0}
    end
  end

  @doc """
  Check if a triple can be inferred (exists in or is entailed by the store).
  """
  @spec entails?(module(), term(), {String.t(), String.t(), String.t()}) :: boolean()
  def entails?(backend, state, {s, p, o}) do
    # Check if it already exists
    {:ok, results} = backend.query(state, subject: s, predicate: p, object: o)

    if results != [] do
      true
    else
      # Materialize into a temporary copy and check
      # For efficiency, we do a full materialize and check
      {:ok, mat_state, _} = materialize(backend, state)
      {:ok, results} = backend.query(mat_state, subject: s, predicate: p, object: o)
      results != []
    end
  end

  # --- Materialization loop ---

  defp materialize_loop(_backend, state, round, max_rounds) when round >= max_rounds do
    Logger.warning("[Reasoner] Hit max rounds limit (#{max_rounds})")
    {:ok, state, round}
  end

  defp materialize_loop(backend, state, round, max_rounds) do
    case apply_rules(backend, state) do
      {:ok, _new_state, 0} ->
        {:ok, state, round}

      {:ok, new_state, n} ->
        Logger.debug("[Reasoner] Round #{round + 1}: #{n} new triples")
        materialize_loop(backend, new_state, round + 1, max_rounds)
    end
  end

  # --- Rule registry ---

  defp rules do
    [
      &rule_rdfs2/2,
      &rule_rdfs3/2,
      &rule_rdfs5/2,
      &rule_rdfs7/2,
      &rule_rdfs9/2,
      &rule_rdfs11/2,
      &rule_prp_eqp/2,
      &rule_prp_inv/2,
      &rule_prp_symp/2,
      &rule_prp_trp/2,
      &rule_cls_hv1/2,
      &rule_cls_hv2/2,
      &rule_cax_eqc/2,
      &rule_eq_sym/2,
      &rule_eq_trans/2,
      &rule_eq_rep/2
    ]
  end

  # --- RDFS Rules ---

  # rdfs2: ?p rdfs:domain ?c, ?x ?p ?y => ?x rdf:type ?c
  defp rule_rdfs2(backend, state) do
    {:ok, domains} = backend.query(state, predicate: @rdfs_domain)

    Enum.flat_map(domains, fn {p, _, c} ->
      {:ok, instances} = backend.query(state, predicate: p)
      Enum.map(instances, fn {x, _, _} -> {x, @rdf_type, c} end)
    end)
  end

  # rdfs3: ?p rdfs:range ?c, ?x ?p ?y => ?y rdf:type ?c
  defp rule_rdfs3(backend, state) do
    {:ok, ranges} = backend.query(state, predicate: @rdfs_range)

    Enum.flat_map(ranges, fn {p, _, c} ->
      {:ok, instances} = backend.query(state, predicate: p)
      Enum.map(instances, fn {_, _, y} -> {y, @rdf_type, c} end)
    end)
  end

  # rdfs5: ?p1 rdfs:subPropertyOf ?p2, ?p2 rdfs:subPropertyOf ?p3 => ?p1 rdfs:subPropertyOf ?p3
  defp rule_rdfs5(backend, state) do
    {:ok, subs} = backend.query(state, predicate: @rdfs_subproperty)

    Enum.flat_map(subs, fn {p1, _, p2} ->
      {:ok, supers} = backend.query(state, subject: p2, predicate: @rdfs_subproperty)
      Enum.map(supers, fn {_, _, p3} -> {p1, @rdfs_subproperty, p3} end)
    end)
  end

  # rdfs7: ?p1 rdfs:subPropertyOf ?p2, ?x ?p1 ?y => ?x ?p2 ?y
  defp rule_rdfs7(backend, state) do
    {:ok, subs} = backend.query(state, predicate: @rdfs_subproperty)

    Enum.flat_map(subs, fn {p1, _, p2} ->
      {:ok, instances} = backend.query(state, predicate: p1)
      Enum.map(instances, fn {x, _, y} -> {x, p2, y} end)
    end)
  end

  # rdfs9: ?c1 rdfs:subClassOf ?c2, ?x rdf:type ?c1 => ?x rdf:type ?c2
  defp rule_rdfs9(backend, state) do
    {:ok, subs} = backend.query(state, predicate: @rdfs_subclass)

    Enum.flat_map(subs, fn {c1, _, c2} ->
      {:ok, instances} = backend.query(state, predicate: @rdf_type, object: c1)
      Enum.map(instances, fn {x, _, _} -> {x, @rdf_type, c2} end)
    end)
  end

  # rdfs11: ?c1 rdfs:subClassOf ?c2, ?c2 rdfs:subClassOf ?c3 => ?c1 rdfs:subClassOf ?c3
  defp rule_rdfs11(backend, state) do
    {:ok, subs} = backend.query(state, predicate: @rdfs_subclass)

    Enum.flat_map(subs, fn {c1, _, c2} ->
      {:ok, supers} = backend.query(state, subject: c2, predicate: @rdfs_subclass)
      Enum.map(supers, fn {_, _, c3} -> {c1, @rdfs_subclass, c3} end)
    end)
  end

  # --- OWL 2 RL Rules ---

  # prp-eqp1/2: ?p1 owl:equivalentProperty ?p2 => mutual rdfs:subPropertyOf
  defp rule_prp_eqp(backend, state) do
    {:ok, eqs} = backend.query(state, predicate: @owl_equivalent_property)

    Enum.flat_map(eqs, fn {p1, _, p2} ->
      [
        {p1, @rdfs_subproperty, p2},
        {p2, @rdfs_subproperty, p1}
      ]
    end)
  end

  # prp-inv1/2: ?p1 owl:inverseOf ?p2, ?x ?p1 ?y => ?y ?p2 ?x (and vice versa)
  defp rule_prp_inv(backend, state) do
    {:ok, inverses} = backend.query(state, predicate: @owl_inverse_of)

    Enum.flat_map(inverses, fn {p1, _, p2} ->
      {:ok, fwd} = backend.query(state, predicate: p1)
      {:ok, rev} = backend.query(state, predicate: p2)

      fwd_inferred = Enum.map(fwd, fn {x, _, y} -> {y, p2, x} end)
      rev_inferred = Enum.map(rev, fn {x, _, y} -> {y, p1, x} end)

      fwd_inferred ++ rev_inferred
    end)
  end

  # prp-symp: ?p rdf:type owl:SymmetricProperty, ?x ?p ?y => ?y ?p ?x
  defp rule_prp_symp(backend, state) do
    {:ok, sym_props} = backend.query(state, predicate: @rdf_type, object: @owl_symmetric)

    Enum.flat_map(sym_props, fn {p, _, _} ->
      {:ok, instances} = backend.query(state, predicate: p)
      Enum.map(instances, fn {x, _, y} -> {y, p, x} end)
    end)
  end

  # prp-trp: ?p rdf:type owl:TransitiveProperty, ?x ?p ?y, ?y ?p ?z => ?x ?p ?z
  defp rule_prp_trp(backend, state) do
    {:ok, trans_props} = backend.query(state, predicate: @rdf_type, object: @owl_transitive)

    Enum.flat_map(trans_props, fn {p, _, _} ->
      {:ok, instances} = backend.query(state, predicate: p)

      # Build adjacency for efficient join
      by_subject =
        Enum.group_by(instances, fn {s, _, _} -> s end, fn {_, _, o} -> o end)

      Enum.flat_map(instances, fn {x, _, y} ->
        case Map.get(by_subject, y) do
          nil -> []
          targets -> Enum.map(targets, fn z -> {x, p, z} end)
        end
      end)
    end)
  end

  # cls-hv1: ?c owl:hasValue ?v, ?c owl:onProperty ?p, ?x rdf:type ?c => ?x ?p ?v
  defp rule_cls_hv1(backend, state) do
    {:ok, hv_triples} = backend.query(state, predicate: @owl_has_value)

    Enum.flat_map(hv_triples, fn {c, _, v} ->
      {:ok, on_prop} = backend.query(state, subject: c, predicate: @owl_on_property)

      Enum.flat_map(on_prop, fn {_, _, p} ->
        {:ok, instances} = backend.query(state, predicate: @rdf_type, object: c)
        Enum.map(instances, fn {x, _, _} -> {x, p, v} end)
      end)
    end)
  end

  # cls-hv2: ?c owl:hasValue ?v, ?c owl:onProperty ?p, ?x ?p ?v => ?x rdf:type ?c
  defp rule_cls_hv2(backend, state) do
    {:ok, hv_triples} = backend.query(state, predicate: @owl_has_value)

    Enum.flat_map(hv_triples, fn {c, _, v} ->
      {:ok, on_prop} = backend.query(state, subject: c, predicate: @owl_on_property)

      Enum.flat_map(on_prop, fn {_, _, p} ->
        {:ok, matches} = backend.query(state, predicate: p, object: v)
        Enum.map(matches, fn {x, _, _} -> {x, @rdf_type, c} end)
      end)
    end)
  end

  # cax-eqc1/2: ?c1 owl:equivalentClass ?c2 => mutual rdfs:subClassOf
  defp rule_cax_eqc(backend, state) do
    {:ok, eqs} = backend.query(state, predicate: @owl_equivalent_class)

    Enum.flat_map(eqs, fn {c1, _, c2} ->
      [
        {c1, @rdfs_subclass, c2},
        {c2, @rdfs_subclass, c1}
      ]
    end)
  end

  # eq-sym: ?x owl:sameAs ?y => ?y owl:sameAs ?x
  defp rule_eq_sym(backend, state) do
    {:ok, sames} = backend.query(state, predicate: @owl_same_as)
    Enum.map(sames, fn {x, _, y} -> {y, @owl_same_as, x} end)
  end

  # eq-trans: ?x owl:sameAs ?y, ?y owl:sameAs ?z => ?x owl:sameAs ?z
  defp rule_eq_trans(backend, state) do
    {:ok, sames} = backend.query(state, predicate: @owl_same_as)

    by_subject =
      Enum.group_by(sames, fn {s, _, _} -> s end, fn {_, _, o} -> o end)

    Enum.flat_map(sames, fn {x, _, y} ->
      case Map.get(by_subject, y) do
        nil -> []
        targets -> Enum.map(targets, fn z -> {x, @owl_same_as, z} end)
      end
    end)
  end

  # eq-rep-s/p/o: sameAs replacement in all positions
  defp rule_eq_rep(backend, state) do
    {:ok, sames} = backend.query(state, predicate: @owl_same_as)

    if sames == [] do
      []
    else
      {:ok, all_triples} = backend.query(state, [])

      Enum.flat_map(sames, fn {x, _, y} ->
        Enum.flat_map(all_triples, fn {s, p, o} ->
          replacements = []

          # Replace in subject position
          replacements =
            if s == x, do: [{y, p, o} | replacements], else: replacements

          # Replace in object position
          replacements =
            if o == x, do: [{s, p, y} | replacements], else: replacements

          # Replace in predicate position
          replacements =
            if p == x, do: [{s, y, o} | replacements], else: replacements

          replacements
        end)
      end)
    end
  end
end
