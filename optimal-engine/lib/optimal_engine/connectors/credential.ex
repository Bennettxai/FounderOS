defmodule OptimalEngine.Connectors.Credential do
  @moduledoc """
  Encrypted-at-rest credential storage for connector auth material.

  Connector config rows live in the `connectors` table. When the config
  contains auth material (OAuth tokens, API keys, private keys) we
  serialize + envelope-encrypt it here before writing, and decrypt
  on read.

  ## Key management

  Phase 9 uses an `AES-256-GCM` key loaded from the `CONNECTOR_KEY`
  env var (hex-encoded 32 bytes). In production the master key should
  come from a KMS / HSM — this module exposes `encrypt/1` + `decrypt/1`
  as the only code paths that touch the key, so swapping providers is
  a one-module change. See `docs/architecture/COMPLIANCE.md` for the
  Phase 11 plan.

  ## Envelope format

      <<version::8, iv::96, tag::128, ciphertext::binary>>

  stored as Base64 in the `config.credentials_ciphertext` JSON field.
  `version = 1` today; any change bumps it and triggers rotation.
  """

  @version 1
  @iv_size 12
  @tag_size 16

  @type plaintext :: map()
  @type envelope :: String.t()

  @doc """
  Encrypt a credentials map. Returns a Base64-encoded envelope or
  `{:error, :key_missing}` when the master key isn't configured.
  """
  @spec encrypt(plaintext()) :: {:ok, envelope()} | {:error, :key_missing}
  def encrypt(creds) when is_map(creds) do
    with {:ok, key} <- master_key() do
      iv = :crypto.strong_rand_bytes(@iv_size)
      plaintext = Jason.encode!(creds)

      {ciphertext, tag} =
        :crypto.crypto_one_time_aead(:aes_256_gcm, key, iv, plaintext, "", true)

      payload = <<@version::8, iv::binary, tag::binary, ciphertext::binary>>
      {:ok, Base.encode64(payload)}
    end
  end

  @doc """
  Decrypt an envelope. Returns the original credentials map or an
  error atom describing what went wrong.
  """
  @spec decrypt(envelope()) ::
          {:ok, plaintext()}
          | {:error, :key_missing | :bad_envelope | :unsupported_version | :auth_failed}
  def decrypt(envelope) when is_binary(envelope) do
    with {:ok, key} <- master_key(),
         {:ok, raw} <- Base.decode64(envelope) |> wrap_bad(),
         {:ok, iv, tag, ciphertext} <- split_envelope(raw),
         {:ok, plaintext} <- aead_decrypt(key, iv, tag, ciphertext),
         {:ok, map} <- Jason.decode(plaintext) do
      {:ok, map}
    end
  end

  @doc """
  Return `true` when the master key is configured. Handy for startup
  health checks so operators know the engine can read existing rows.
  """
  @spec ready?() :: boolean()
  def ready? do
    match?({:ok, _}, master_key())
  end

  # ─── private ─────────────────────────────────────────────────────────────

  defp master_key do
    case System.get_env("CONNECTOR_KEY") do
      nil ->
        {:error, :key_missing}

      hex when byte_size(hex) == 64 ->
        case Base.decode16(hex, case: :mixed) do
          {:ok, key} when byte_size(key) == 32 -> {:ok, key}
          _ -> {:error, :key_missing}
        end

      _ ->
        {:error, :key_missing}
    end
  end

  defp wrap_bad(:error), do: {:error, :bad_envelope}
  defp wrap_bad({:ok, _} = ok), do: ok

  defp split_envelope(
         <<@version::8, iv::binary-size(@iv_size), tag::binary-size(@tag_size), rest::binary>>
       ) do
    {:ok, iv, tag, rest}
  end

  defp split_envelope(<<v::8, _::binary>>) when v != @version, do: {:error, :unsupported_version}
  defp split_envelope(_), do: {:error, :bad_envelope}

  defp aead_decrypt(key, iv, tag, ciphertext) do
    case :crypto.crypto_one_time_aead(:aes_256_gcm, key, iv, ciphertext, "", tag, false) do
      plaintext when is_binary(plaintext) -> {:ok, plaintext}
      _ -> {:error, :auth_failed}
    end
  end
end
