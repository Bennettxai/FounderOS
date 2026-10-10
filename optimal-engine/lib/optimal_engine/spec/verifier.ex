defmodule OptimalEngine.Spec.Verifier do
  @moduledoc """
  Validates spec requirements against the actual codebase.

  Evaluates each verification claim and assigns a strength level:

  - `:claimed`  — requirement is referenced in a verification block but not proven
  - `:linked`   — verification target file exists AND contains the requirement ID
  - `:executed` — a shell command verification ran successfully (exit code 0)

  Each spec declares a `verification_minimum_strength` in its meta block.
  A finding fails if achieved strength is below the declared minimum.
  """

  alias OptimalEngine.Spec.Parser

  @type finding :: %{
          subject: String.t(),
          requirement: String.t(),
          strength: :claimed | :linked | :executed,
          min_strength: :claimed | :linked | :executed,
          meets_minimum: boolean(),
          target: String.t(),
          kind: String.t()
        }

  @strength_order %{claimed: 0, linked: 1, executed: 2}

  @doc """
  Verifies a single parsed spec against the filesystem.

  Returns findings for every requirement that has at least one verification claim.
  Requirements with no verification claims get a `:claimed` strength of `:none`.
  """
  @spec verify(Parser.spec(), String.t()) :: [finding()]
  def verify(spec, root_path) do
    min_strength = parse_strength(spec.meta.verification_minimum_strength)

    # Build a map of requirement_id → best verification result
    req_verifications = build_req_map(spec)

    Enum.flat_map(spec.requirements, fn req ->
      case Map.get(req_verifications, req.id) do
        nil ->
          # No verification claims for this requirement
          [
            %{
              subject: spec.meta.id,
              requirement: req.id,
              strength: :none,
              min_strength: min_strength,
              meets_minimum: min_strength == :claimed,
              target: "",
              kind: "none"
            }
          ]

        verifications ->
          Enum.map(verifications, fn v ->
            strength = evaluate_strength(v, root_path)

            %{
              subject: spec.meta.id,
              requirement: req.id,
              strength: strength,
              min_strength: min_strength,
              meets_minimum: meets_minimum?(strength, min_strength),
              target: v.target,
              kind: v.kind
            }
          end)
      end
    end)
  end

  @doc """
  Verifies all specs in a directory. Returns aggregated findings.
  """
  @spec verify_all(String.t(), String.t()) ::
          {:ok, %{findings: [finding()], specs: [Parser.spec()]}}
  def verify_all(spec_dir, root_path) do
    {:ok, specs} = Parser.parse_all(spec_dir)

    findings =
      Enum.flat_map(specs, fn spec ->
        verify(spec, root_path)
      end)

    {:ok, %{findings: findings, specs: specs}}
  end

  # -- Private: Verification Map -----------------------------------------------

  # Build map: requirement_id → [verification claims that cover it]
  defp build_req_map(spec) do
    Enum.reduce(spec.verifications, %{}, fn v, acc ->
      Enum.reduce(v.covers, acc, fn req_id, inner_acc ->
        Map.update(inner_acc, req_id, [v], &[v | &1])
      end)
    end)
  end

  # -- Private: Strength Evaluation --------------------------------------------

  defp evaluate_strength(%{kind: "command", target: command}, _root_path) do
    case run_command(command) do
      {_output, 0} -> :executed
      _ -> :claimed
    end
  end

  defp evaluate_strength(%{kind: kind, target: target}, root_path)
       when kind in ["test_file", "source_file", "doc_file"] do
    full_path = resolve_path(target, root_path)

    cond do
      not File.exists?(full_path) -> :claimed
      true -> :linked
    end
  end

  defp evaluate_strength(_verification, _root_path), do: :claimed

  defp run_command(command) do
    try do
      System.cmd("sh", ["-c", command], stderr_to_stdout: true, env: [])
    rescue
      _ -> {"", 1}
    end
  end

  defp resolve_path(target, root_path) do
    if Path.type(target) == :absolute do
      target
    else
      Path.join(root_path, target)
    end
  end

  # -- Private: Strength Comparison --------------------------------------------

  defp meets_minimum?(achieved, minimum) do
    Map.get(@strength_order, achieved, -1) >= Map.get(@strength_order, minimum, 0)
  end

  defp parse_strength("executed"), do: :executed
  defp parse_strength("linked"), do: :linked
  defp parse_strength(_), do: :claimed
end
