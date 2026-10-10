defmodule OptimalEngine.Connectors.CredentialTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Connectors.Credential

  @valid_key Base.encode16(:crypto.strong_rand_bytes(32), case: :lower)

  setup do
    original = System.get_env("CONNECTOR_KEY")

    on_exit(fn ->
      if original,
        do: System.put_env("CONNECTOR_KEY", original),
        else: System.delete_env("CONNECTOR_KEY")
    end)

    :ok
  end

  describe "encrypt/1 + decrypt/1 round-trip" do
    test "plaintext comes back identical" do
      System.put_env("CONNECTOR_KEY", @valid_key)
      creds = %{"bot_token" => "xoxb-abc", "scope" => "channels:history"}

      assert {:ok, envelope} = Credential.encrypt(creds)
      assert is_binary(envelope)
      assert {:ok, ^creds} = Credential.decrypt(envelope)
    end

    test "ciphertext differs across two calls for the same plaintext (IV randomness)" do
      System.put_env("CONNECTOR_KEY", @valid_key)
      creds = %{"k" => "v"}

      {:ok, e1} = Credential.encrypt(creds)
      {:ok, e2} = Credential.encrypt(creds)

      refute e1 == e2
    end
  end

  describe "error cases" do
    test "missing key returns :key_missing on encrypt" do
      System.delete_env("CONNECTOR_KEY")
      assert {:error, :key_missing} = Credential.encrypt(%{"x" => 1})
    end

    test "missing key returns :key_missing on decrypt" do
      System.delete_env("CONNECTOR_KEY")
      assert {:error, :key_missing} = Credential.decrypt("not-a-real-envelope")
    end

    test "invalid envelope rejected" do
      System.put_env("CONNECTOR_KEY", @valid_key)
      assert {:error, :bad_envelope} = Credential.decrypt("notbase64!!!")
    end

    test "tampered envelope fails auth" do
      System.put_env("CONNECTOR_KEY", @valid_key)
      {:ok, envelope} = Credential.encrypt(%{"x" => "y"})

      # Flip a middle byte by decoding, mutating, re-encoding.
      bad =
        envelope
        |> Base.decode64!()
        |> then(fn <<head::binary-size(30), byte, rest::binary>> ->
          <<head::binary, byte + 1, rest::binary>>
        end)
        |> Base.encode64()

      assert {:error, :auth_failed} = Credential.decrypt(bad)
    end
  end

  describe "ready?/0" do
    test "true when key set, false otherwise" do
      System.put_env("CONNECTOR_KEY", @valid_key)
      assert Credential.ready?()

      System.delete_env("CONNECTOR_KEY")
      refute Credential.ready?()
    end
  end
end
