defmodule OptimalEngine.MemoryCore.ScoringPolicy do
  @moduledoc """
  Versioned confidence and precision scoring for the Truth Lifecycle.

  Confidence (is it true?) and precision (how specific is it?) are SEPARATE
  scores and are never collapsed into one number. Every score this policy
  produces is recorded with `version/0` in the derivation ledger so later
  calibration can replay or re-score.

  The policy also owns the evidence-review decision between Claim and Fact:
  a Claim is promoted only through explicit actor approval (`:verifier_id` or
  `:actor_id`) or an opt-in auto-promotion policy (`policy: :auto`) gated by
  `auto_promote_threshold/0`. The threshold is fixed by the policy version
  and cannot be overridden per call, and a reviewer may not approve a Claim
  they evaluated themselves unless the caller explicitly opts in with
  `allow_self_review: true`. Summaries, search hits, and observations never
  become Facts automatically.
  """

  alias OptimalEngine.MemoryCore.{Claim, Fact, SourcePackage}

  @policy_version "memory_core.scoring_policy.v1"
  @auto_promote_threshold 0.85

  # Confidence baseline by source trust label. "unreviewed" intentionally
  # matches the historical 0.55 extraction default.
  @trust_confidence %{
    "verified" => 0.75,
    "reviewed" => 0.65,
    "unreviewed" => 0.55,
    "quarantined" => 0.35
  }
  @default_confidence 0.55

  # Precision baseline plus a bonus per structured anchor supplied
  # (subject/action/object). With no anchors this matches the historical
  # 0.55 extraction default.
  @base_precision 0.55
  @anchor_precision_bonus 0.05

  @spec version() :: String.t()
  def version, do: @policy_version

  @spec auto_promote_threshold() :: float()
  def auto_promote_threshold, do: @auto_promote_threshold

  @doc """
  Per-workspace auto-promotion confidence threshold.

  The policy-version floor (`auto_promote_threshold/0`) still governs the
  Claim→Fact review decision; this helper lets the autonomous promotion
  scheduler require a workspace-specific minimum confidence before it offers a
  Claim for `policy: :auto` promotion. Resolution order:

    1. `config :optimal_engine, :memory, auto_promote_thresholds: %{workspace_id => float}`
    2. `config :optimal_engine, :memory, auto_promote_threshold: float`
    3. the policy-version floor (`auto_promote_threshold/0`)

  The returned value is never below the policy floor.
  """
  @spec auto_promote_threshold(String.t() | nil) :: float()
  def auto_promote_threshold(workspace_id) do
    memory_config = Application.get_env(:optimal_engine, :memory, [])
    per_workspace = Keyword.get(memory_config, :auto_promote_thresholds, %{})

    configured =
      Map.get(per_workspace, to_string(workspace_id)) ||
        Keyword.get(memory_config, :auto_promote_threshold) ||
        @auto_promote_threshold

    max(clamp(configured), @auto_promote_threshold)
  end

  @doc """
  Score a Claim extracted from a Source Package.

  Caller-supplied `:aggregate_confidence`/`:aggregate_precision` win (the
  extractor may know more than the policy); otherwise confidence derives from
  the source trust label and precision from anchor specificity.
  """
  @spec claim_scores(SourcePackage.t(), keyword()) :: %{
          confidence: float(),
          precision: float(),
          raw_component_scores: map(),
          calibration_dataset_version: String.t() | nil
        }
  def claim_scores(%SourcePackage{} = source_package, opts \\ []) do
    anchor_count =
      [
        Keyword.get(opts, :subject_anchor),
        Keyword.get(opts, :action_class),
        Keyword.get(opts, :object_anchor)
      ]
      |> Enum.count(&present?/1)

    base_confidence = Map.get(@trust_confidence, source_package.trust_label, @default_confidence)
    base_precision = clamp(@base_precision + anchor_count * @anchor_precision_bonus)

    confidence = clamp(Keyword.get(opts, :aggregate_confidence) || base_confidence)
    precision = clamp(Keyword.get(opts, :aggregate_precision) || base_precision)

    %{
      confidence: confidence,
      precision: precision,
      raw_component_scores:
        Keyword.get(opts, :raw_component_scores) ||
          %{
            "policy_version" => @policy_version,
            "source_trust_label" => source_package.trust_label,
            "trust_base_confidence" => base_confidence,
            "anchor_count" => anchor_count,
            "anchor_base_precision" => base_precision,
            "caller_override" =>
              Keyword.has_key?(opts, :aggregate_confidence) or
                Keyword.has_key?(opts, :aggregate_precision)
          },
      calibration_dataset_version: Keyword.get(opts, :calibration_dataset_version)
    }
  end

  @doc """
  Score a Fact promoted from a Claim.

  Promotion itself does not manufacture confidence: scores inherit from the
  Claim unless the reviewer overrides them. Deltas against the Claim are
  returned for the derivation ledger.
  """
  @spec promotion_scores(Claim.t(), keyword()) :: %{
          confidence: float(),
          precision: float(),
          confidence_delta: float(),
          precision_delta: float(),
          raw_component_scores: map()
        }
  def promotion_scores(%Claim{} = claim, opts \\ []) do
    confidence = clamp(Keyword.get(opts, :aggregate_confidence) || claim.aggregate_confidence)
    precision = clamp(Keyword.get(opts, :aggregate_precision) || claim.aggregate_precision)

    %{
      confidence: confidence,
      precision: precision,
      confidence_delta: confidence - claim.aggregate_confidence,
      precision_delta: precision - claim.aggregate_precision,
      raw_component_scores:
        Keyword.get(opts, :raw_component_scores) ||
          %{
            "policy_version" => @policy_version,
            "claim_confidence" => claim.aggregate_confidence,
            "claim_precision" => claim.aggregate_precision,
            "reviewer_override" =>
              Keyword.has_key?(opts, :aggregate_confidence) or
                Keyword.has_key?(opts, :aggregate_precision)
          }
    }
  end

  @doc """
  Score a Memory Object built around an accepted Fact (inherits unless
  overridden; salience is a separate usefulness signal, not confidence).
  """
  @spec memory_scores(Fact.t(), keyword()) :: %{
          confidence: float(),
          precision: float(),
          salience: float()
        }
  def memory_scores(%Fact{} = fact, opts \\ []) do
    %{
      confidence: clamp(Keyword.get(opts, :aggregate_confidence) || fact.aggregate_confidence),
      precision: clamp(Keyword.get(opts, :aggregate_precision) || fact.aggregate_precision),
      salience: clamp(Keyword.get(opts, :salience) || 0.5)
    }
  end

  @doc """
  Evidence-review decision for promoting a Claim to a Fact.

  The auto-promotion threshold is fixed by this policy version
  (`auto_promote_threshold/0`); callers cannot override it per call.
  Self-review is rejected: a verifier matching the Claim's own
  `evaluator_id` (the extracting/evaluating identity) is refused unless the
  caller explicitly passes `allow_self_review: true`.

  Returns `{:ok, review}` when promotion is approved. `review` carries
  `:verifier_id` and `:review_basis` plus the `:effective_threshold` and
  `:claim_confidence` that produced the decision, so the promotion ledger
  entry can record them for replay. Otherwise an error:

    * `{:error, :claim_rejected}` - the claim was already rejected in review
    * `{:error, :claim_already_promoted}` - the claim was already promoted
    * `{:error, :self_review_not_allowed}` - the verifier is the claim's own
      `evaluator_id` and `allow_self_review: true` was not given
    * `{:error, :below_auto_promote_threshold}` - `policy: :auto` was given
      but the claim confidence is below `auto_promote_threshold/0`
    * `{:error, :approval_required}` - no actor approval and no policy opt-in
  """
  @spec review_decision(Claim.t(), keyword()) ::
          {:ok,
           %{
             verifier_id: String.t(),
             review_basis: String.t(),
             effective_threshold: float(),
             claim_confidence: float()
           }}
          | {:error, term()}
  def review_decision(%Claim{} = claim, opts \\ []) do
    verifier = string_or_nil(Keyword.get(opts, :verifier_id) || Keyword.get(opts, :actor_id))

    cond do
      claim.lifecycle_state == "rejected" or claim.review_status == "rejected" ->
        {:error, :claim_rejected}

      claim.lifecycle_state == "promoted" ->
        {:error, :claim_already_promoted}

      self_review?(claim, verifier) and Keyword.get(opts, :allow_self_review, false) != true ->
        {:error, :self_review_not_allowed}

      present?(verifier) ->
        {:ok, approval(verifier, "actor_approval", claim)}

      Keyword.get(opts, :policy) == :auto and
          claim.aggregate_confidence >= @auto_promote_threshold ->
        {:ok, approval("policy:#{@policy_version}", "policy_auto_promotion", claim)}

      Keyword.get(opts, :policy) == :auto ->
        {:error, :below_auto_promote_threshold}

      true ->
        {:error, :approval_required}
    end
  end

  defp approval(verifier_id, review_basis, %Claim{} = claim) do
    %{
      verifier_id: verifier_id,
      review_basis: review_basis,
      effective_threshold: @auto_promote_threshold,
      claim_confidence: claim.aggregate_confidence
    }
  end

  defp self_review?(%Claim{evaluator_id: evaluator_id}, verifier) do
    present?(verifier) and present?(evaluator_id) and to_string(evaluator_id) == verifier
  end

  defp present?(nil), do: false
  defp present?(""), do: false
  defp present?(_), do: true

  defp string_or_nil(nil), do: nil
  defp string_or_nil(""), do: nil
  defp string_or_nil(value), do: to_string(value)

  defp clamp(score) when is_number(score), do: score |> max(0.0) |> min(1.0) |> to_float()

  defp to_float(score) when is_integer(score), do: score / 1
  defp to_float(score), do: score
end
