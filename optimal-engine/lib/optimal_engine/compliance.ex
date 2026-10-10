defmodule OptimalEngine.Compliance do
  @moduledoc """
  Top-level facade for the compliance layer.

  Every public function routes to a focused sub-module:

  | Call                                           | Sub-module                       |
  |-----------------------------------------------|----------------------------------|
  | `scan_pii(text)`                              | `Compliance.PII`                 |
  | `redact(text, opts)`                          | `Compliance.Redact`              |
  | `dsar_export(principal_id)`                   | `Compliance.DSAR`                |
  | `erase(principal_id, opts)`                   | `Compliance.Erasure`             |
  | `erasure_preview(principal_id, opts)`         | `Compliance.Erasure`             |
  | `place_legal_hold(signal_id, who, reason)`    | `Compliance.LegalHold`           |
  | `release_legal_hold(hold_id)`                 | `Compliance.LegalHold`           |
  | `retention_sweep(opts)`                       | `Compliance.Retention`           |

  Regulatory frames covered:
    * GDPR (Art. 15 → DSAR, Art. 17 → Erasure, Art. 5(1)(e) → Retention)
    * CCPA / CPRA (§1798.110, §1798.105)
    * HIPAA (§164.524, §164.526, §164.530(c))
    * SOC 2 (CC 6.7 audit trail via `events`, CC 4 retention)

  See `docs/architecture/COMPLIANCE.md` for the control-to-code map.
  """

  alias OptimalEngine.Compliance.{DSAR, Erasure, LegalHold, PII, Redact, Retention}

  defdelegate scan_pii(text), to: PII, as: :scan

  @doc "Redact PII in `text`. See `Compliance.Redact.redact/2`."
  def redact(text, opts \\ []), do: Redact.redact(text, opts)

  @doc "Export every record tied to `principal_id` (GDPR Art. 15)."
  def dsar_export(principal_id, tenant_id \\ "default") do
    DSAR.export(principal_id, tenant_id)
  end

  @doc "Erase every record tied to `principal_id` (GDPR Art. 17)."
  def erase(principal_id, opts \\ []), do: Erasure.erase(principal_id, opts)

  @doc "Dry run — what would an erasure delete?"
  def erasure_preview(principal_id, opts \\ []), do: Erasure.preview(principal_id, opts)

  @doc "Place a legal hold on a signal."
  def place_legal_hold(signal_id, held_by, reason, opts \\ []) do
    LegalHold.place(signal_id, held_by, reason, opts)
  end

  @doc "Release a legal hold by id."
  defdelegate release_legal_hold(hold_id), to: LegalHold, as: :release

  @doc "List active legal holds for a tenant."
  def active_legal_holds(tenant_id \\ "default"), do: LegalHold.active(tenant_id)

  @doc "Apply retention policies. See `Compliance.Retention.sweep/1`."
  def retention_sweep(opts \\ []), do: Retention.sweep(opts)

  @doc "Register a retention policy. See `Compliance.Retention.add_policy/1`."
  def add_retention_policy(attrs), do: Retention.add_policy(attrs)
end
