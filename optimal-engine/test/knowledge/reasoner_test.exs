defmodule OptimalEngine.Knowledge.ReasonerTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Knowledge.Backend.ETS
  alias OptimalEngine.Knowledge.Reasoner

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

  setup do
    id = "test_#{:erlang.unique_integer([:positive])}"
    {:ok, state} = ETS.init(id, [])
    on_exit(fn -> ETS.terminate(state) end)
    %{state: state}
  end

  defp assert_triple(state, s, p, o) do
    {:ok, state} = ETS.assert(state, s, p, o)
    state
  end

  defp has_triple?(state, s, p, o) do
    {:ok, results} = ETS.query(state, subject: s, predicate: p, object: o)
    results != []
  end

  # --- 1. RDFS subClassOf inference ---

  describe "RDFS subClassOf inference (rdfs9 + rdfs11)" do
    test "should infer transitive type membership through class hierarchy", %{state: state} do
      # Animal > Mammal > Dog
      state = assert_triple(state, "ex:Dog", @rdfs_subclass, "ex:Mammal")
      state = assert_triple(state, "ex:Mammal", @rdfs_subclass, "ex:Animal")
      state = assert_triple(state, "ex:fido", @rdf_type, "ex:Dog")

      {:ok, state, _rounds} = Reasoner.materialize(ETS, state)

      # fido should be inferred as Mammal and Animal
      assert has_triple?(state, "ex:fido", @rdf_type, "ex:Mammal")
      assert has_triple?(state, "ex:fido", @rdf_type, "ex:Animal")

      # Transitive subClassOf: Dog subClassOf Animal
      assert has_triple?(state, "ex:Dog", @rdfs_subclass, "ex:Animal")
    end
  end

  # --- 2. RDFS subPropertyOf ---

  describe "RDFS subPropertyOf inference (rdfs7)" do
    test "should infer parent property usage from sub-property", %{state: state} do
      state = assert_triple(state, "ex:hasMother", @rdfs_subproperty, "ex:hasParent")
      state = assert_triple(state, "ex:alice", "ex:hasMother", "ex:carol")

      {:ok, state, _rounds} = Reasoner.materialize(ETS, state)

      assert has_triple?(state, "ex:alice", "ex:hasParent", "ex:carol")
    end

    test "should infer through transitive subPropertyOf chain", %{state: state} do
      state = assert_triple(state, "ex:hasMother", @rdfs_subproperty, "ex:hasParent")
      state = assert_triple(state, "ex:hasParent", @rdfs_subproperty, "ex:hasAncestor")
      state = assert_triple(state, "ex:alice", "ex:hasMother", "ex:carol")

      {:ok, state, _rounds} = Reasoner.materialize(ETS, state)

      # hasMother subPropertyOf hasAncestor (transitive)
      assert has_triple?(state, "ex:hasMother", @rdfs_subproperty, "ex:hasAncestor")
      # alice hasAncestor carol (through chain)
      assert has_triple?(state, "ex:alice", "ex:hasAncestor", "ex:carol")
    end
  end

  # --- 3. RDFS domain/range ---

  describe "RDFS domain/range inference (rdfs2 + rdfs3)" do
    test "should infer subject type from property domain", %{state: state} do
      state = assert_triple(state, "ex:teaches", @rdfs_domain, "ex:Professor")
      state = assert_triple(state, "ex:alice", "ex:teaches", "ex:cs101")

      {:ok, state, _rounds} = Reasoner.materialize(ETS, state)

      assert has_triple?(state, "ex:alice", @rdf_type, "ex:Professor")
    end

    test "should infer object type from property range", %{state: state} do
      state = assert_triple(state, "ex:teaches", @rdfs_range, "ex:Course")
      state = assert_triple(state, "ex:alice", "ex:teaches", "ex:cs101")

      {:ok, state, _rounds} = Reasoner.materialize(ETS, state)

      assert has_triple?(state, "ex:cs101", @rdf_type, "ex:Course")
    end

    test "should infer both domain and range types", %{state: state} do
      state = assert_triple(state, "ex:teaches", @rdfs_domain, "ex:Professor")
      state = assert_triple(state, "ex:teaches", @rdfs_range, "ex:Course")
      state = assert_triple(state, "ex:alice", "ex:teaches", "ex:cs101")

      {:ok, state, _rounds} = Reasoner.materialize(ETS, state)

      assert has_triple?(state, "ex:alice", @rdf_type, "ex:Professor")
      assert has_triple?(state, "ex:cs101", @rdf_type, "ex:Course")
    end
  end

  # --- 4. OWL TransitiveProperty ---

  describe "OWL TransitiveProperty (prp-trp)" do
    test "should infer transitive closure", %{state: state} do
      state = assert_triple(state, "ex:ancestor", @rdf_type, @owl_transitive)
      state = assert_triple(state, "ex:alice", "ex:ancestor", "ex:bob")
      state = assert_triple(state, "ex:bob", "ex:ancestor", "ex:carol")
      state = assert_triple(state, "ex:carol", "ex:ancestor", "ex:dave")

      {:ok, state, _rounds} = Reasoner.materialize(ETS, state)

      assert has_triple?(state, "ex:alice", "ex:ancestor", "ex:carol")
      assert has_triple?(state, "ex:alice", "ex:ancestor", "ex:dave")
      assert has_triple?(state, "ex:bob", "ex:ancestor", "ex:dave")
    end
  end

  # --- 5. OWL SymmetricProperty ---

  describe "OWL SymmetricProperty (prp-symp)" do
    test "should infer reverse direction for symmetric property", %{state: state} do
      state = assert_triple(state, "ex:friendOf", @rdf_type, @owl_symmetric)
      state = assert_triple(state, "ex:alice", "ex:friendOf", "ex:bob")

      {:ok, state, _rounds} = Reasoner.materialize(ETS, state)

      assert has_triple?(state, "ex:bob", "ex:friendOf", "ex:alice")
    end
  end

  # --- 6. OWL inverseOf ---

  describe "OWL inverseOf (prp-inv)" do
    test "should infer inverse property", %{state: state} do
      state = assert_triple(state, "ex:parentOf", @owl_inverse_of, "ex:childOf")
      state = assert_triple(state, "ex:alice", "ex:parentOf", "ex:bob")

      {:ok, state, _rounds} = Reasoner.materialize(ETS, state)

      assert has_triple?(state, "ex:bob", "ex:childOf", "ex:alice")
    end

    test "should infer in both directions", %{state: state} do
      state = assert_triple(state, "ex:parentOf", @owl_inverse_of, "ex:childOf")
      state = assert_triple(state, "ex:bob", "ex:childOf", "ex:alice")

      {:ok, state, _rounds} = Reasoner.materialize(ETS, state)

      assert has_triple?(state, "ex:alice", "ex:parentOf", "ex:bob")
    end
  end

  # --- 7. OWL sameAs ---

  describe "OWL sameAs (eq-sym, eq-trans, eq-rep)" do
    test "should infer symmetric sameAs", %{state: state} do
      state = assert_triple(state, "ex:a", @owl_same_as, "ex:b")

      {:ok, state, _rounds} = Reasoner.materialize(ETS, state)

      assert has_triple?(state, "ex:b", @owl_same_as, "ex:a")
    end

    test "should infer transitive sameAs", %{state: state} do
      state = assert_triple(state, "ex:a", @owl_same_as, "ex:b")
      state = assert_triple(state, "ex:b", @owl_same_as, "ex:c")

      {:ok, state, _rounds} = Reasoner.materialize(ETS, state)

      assert has_triple?(state, "ex:a", @owl_same_as, "ex:c")
    end

    test "should replace sameAs individuals in triples", %{state: state} do
      state = assert_triple(state, "ex:a", @owl_same_as, "ex:b")
      state = assert_triple(state, "ex:a", "ex:likes", "ex:pizza")

      {:ok, state, _rounds} = Reasoner.materialize(ETS, state)

      # b should also like pizza (subject replacement)
      assert has_triple?(state, "ex:b", "ex:likes", "ex:pizza")
    end

    test "should replace sameAs in object position", %{state: state} do
      state = assert_triple(state, "ex:a", @owl_same_as, "ex:b")
      state = assert_triple(state, "ex:charlie", "ex:knows", "ex:a")

      {:ok, state, _rounds} = Reasoner.materialize(ETS, state)

      assert has_triple?(state, "ex:charlie", "ex:knows", "ex:b")
    end
  end

  # --- 8. OWL equivalentClass ---

  describe "OWL equivalentClass (cax-eqc)" do
    test "should infer mutual subClassOf from equivalentClass", %{state: state} do
      state = assert_triple(state, "ex:Car", @owl_equivalent_class, "ex:Automobile")

      {:ok, state, _rounds} = Reasoner.materialize(ETS, state)

      assert has_triple?(state, "ex:Car", @rdfs_subclass, "ex:Automobile")
      assert has_triple?(state, "ex:Automobile", @rdfs_subclass, "ex:Car")
    end

    test "should propagate types through equivalentClass", %{state: state} do
      state = assert_triple(state, "ex:Car", @owl_equivalent_class, "ex:Automobile")
      state = assert_triple(state, "ex:myTesla", @rdf_type, "ex:Car")

      {:ok, state, _rounds} = Reasoner.materialize(ETS, state)

      # myTesla should also be an Automobile via subClassOf + rdfs9
      assert has_triple?(state, "ex:myTesla", @rdf_type, "ex:Automobile")
    end
  end

  # --- 9. OWL hasValue restriction ---

  describe "OWL hasValue restriction (cls-hv1 + cls-hv2)" do
    test "should infer property value from type membership (cls-hv1)", %{state: state} do
      # Restriction: AustralianThing hasValue "Australia" onProperty country
      state = assert_triple(state, "ex:AustralianThing", @owl_has_value, "ex:Australia")
      state = assert_triple(state, "ex:AustralianThing", @owl_on_property, "ex:country")
      state = assert_triple(state, "ex:sydney", @rdf_type, "ex:AustralianThing")

      {:ok, state, _rounds} = Reasoner.materialize(ETS, state)

      assert has_triple?(state, "ex:sydney", "ex:country", "ex:Australia")
    end

    test "should infer type from property value (cls-hv2)", %{state: state} do
      state = assert_triple(state, "ex:AustralianThing", @owl_has_value, "ex:Australia")
      state = assert_triple(state, "ex:AustralianThing", @owl_on_property, "ex:country")
      state = assert_triple(state, "ex:sydney", "ex:country", "ex:Australia")

      {:ok, state, _rounds} = Reasoner.materialize(ETS, state)

      assert has_triple?(state, "ex:sydney", @rdf_type, "ex:AustralianThing")
    end
  end

  # --- 10. Fixed-point convergence ---

  describe "fixed-point convergence" do
    test "should terminate and return round count", %{state: state} do
      state = assert_triple(state, "ex:Dog", @rdfs_subclass, "ex:Mammal")
      state = assert_triple(state, "ex:Mammal", @rdfs_subclass, "ex:Animal")
      state = assert_triple(state, "ex:Animal", @rdfs_subclass, "ex:LivingThing")
      state = assert_triple(state, "ex:fido", @rdf_type, "ex:Dog")

      {:ok, state, rounds} = Reasoner.materialize(ETS, state)

      # Should converge in finite rounds
      assert rounds > 0
      assert rounds < 100

      # All inferences present
      assert has_triple?(state, "ex:fido", @rdf_type, "ex:Mammal")
      assert has_triple?(state, "ex:fido", @rdf_type, "ex:Animal")
      assert has_triple?(state, "ex:fido", @rdf_type, "ex:LivingThing")
    end

    test "should respect max_rounds limit", %{state: state} do
      state = assert_triple(state, "ex:Dog", @rdfs_subclass, "ex:Mammal")
      state = assert_triple(state, "ex:fido", @rdf_type, "ex:Dog")

      {:ok, _state, rounds} = Reasoner.materialize(ETS, state, max_rounds: 1)

      assert rounds <= 1
    end
  end

  # --- 11. Empty store ---

  describe "empty store" do
    test "should return 0 rounds on empty store", %{state: state} do
      {:ok, _state, rounds} = Reasoner.materialize(ETS, state)
      assert rounds == 0
    end
  end

  # --- 12. entails? ---

  describe "entails?/3" do
    test "should return true for explicitly asserted triple", %{state: state} do
      state = assert_triple(state, "ex:a", "ex:b", "ex:c")
      assert Reasoner.entails?(ETS, state, {"ex:a", "ex:b", "ex:c"})
    end

    test "should return true for inferable triple", %{state: state} do
      state = assert_triple(state, "ex:Dog", @rdfs_subclass, "ex:Animal")
      state = assert_triple(state, "ex:fido", @rdf_type, "ex:Dog")

      assert Reasoner.entails?(ETS, state, {"ex:fido", @rdf_type, "ex:Animal"})
    end

    test "should return false for non-entailed triple", %{state: state} do
      state = assert_triple(state, "ex:fido", @rdf_type, "ex:Dog")
      refute Reasoner.entails?(ETS, state, {"ex:fido", @rdf_type, "ex:Cat"})
    end
  end

  # --- 13. equivalentProperty ---

  describe "OWL equivalentProperty (prp-eqp)" do
    test "should infer mutual subPropertyOf", %{state: state} do
      state = assert_triple(state, "ex:cost", @owl_equivalent_property, "ex:price")

      {:ok, state, _rounds} = Reasoner.materialize(ETS, state)

      assert has_triple?(state, "ex:cost", @rdfs_subproperty, "ex:price")
      assert has_triple?(state, "ex:price", @rdfs_subproperty, "ex:cost")
    end

    test "should propagate property usage through equivalence", %{state: state} do
      state = assert_triple(state, "ex:cost", @owl_equivalent_property, "ex:price")
      state = assert_triple(state, "ex:widget", "ex:cost", "100")

      {:ok, state, _rounds} = Reasoner.materialize(ETS, state)

      # Through equivalentProperty -> subPropertyOf -> rdfs7
      assert has_triple?(state, "ex:widget", "ex:price", "100")
    end
  end
end
