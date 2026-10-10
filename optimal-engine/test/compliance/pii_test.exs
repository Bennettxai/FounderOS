defmodule OptimalEngine.Compliance.PIITest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Compliance.PII

  describe "scan/1" do
    test "detects email addresses" do
      text = "Ping me at alice@example.com or bob.jr+work@sample.co.uk"
      matches = PII.scan(text)
      kinds = Enum.map(matches, & &1.kind)
      assert :email in kinds

      emails = matches |> Enum.filter(&(&1.kind == :email)) |> Enum.map(& &1.value)
      assert "alice@example.com" in emails
      assert "bob.jr+work@sample.co.uk" in emails
    end

    test "detects US SSNs" do
      assert PII.scan("SSN is 123-45-6789.") |> Enum.any?(&(&1.kind == :ssn))
    end

    test "rejects obvious SSN lookalikes (000, 666, 9xx areas)" do
      refute PII.scan("not real 000-12-3456") |> Enum.any?(&(&1.kind == :ssn))
      refute PII.scan("not real 666-12-3456") |> Enum.any?(&(&1.kind == :ssn))
      refute PII.scan("not real 912-12-3456") |> Enum.any?(&(&1.kind == :ssn))
    end

    test "detects IPv4 addresses but not bogus octets" do
      assert PII.scan("Server at 192.168.1.10.") |> Enum.any?(&(&1.kind == :ipv4))
      refute PII.scan("Version 999.1.1.1") |> Enum.any?(&(&1.kind == :ipv4))
    end

    test "detects URLs" do
      assert PII.scan("See https://example.com/path?x=1") |> Enum.any?(&(&1.kind == :url))
    end

    test "detects valid credit card numbers via Luhn" do
      # Canonical test VISA number that passes Luhn
      assert PII.scan("Card 4532015112830366.") |> Enum.any?(&(&1.kind == :credit_card))
    end

    test "rejects Luhn-invalid digit strings" do
      refute PII.scan("Not a card 1234567890123456.") |> Enum.any?(&(&1.kind == :credit_card))
    end

    test "detects NANP phone numbers" do
      assert PII.scan("Call (555) 123-4567.") |> Enum.any?(&(&1.kind == :phone))
      assert PII.scan("Reach me at +1 555-123-4567.") |> Enum.any?(&(&1.kind == :phone))
    end

    test "returns matches in document order" do
      text = "mail: x@y.com ip: 10.0.0.1"
      matches = PII.scan(text)
      assert matches == Enum.sort_by(matches, & &1.offset)
    end

    test "any? mirrors scan/1 presence" do
      assert PII.any?("email: x@y.com")
      refute PII.any?("plain text nothing sensitive")
    end

    test "kinds_present returns deduplicated kinds" do
      text = "a@b.com and c@d.com"
      assert PII.kinds_present(text) == [:email]
    end
  end
end
