defmodule OptimalEngine.Compliance.RedactTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Compliance.Redact

  describe "redact/2 — :placeholder (default)" do
    test "substitutes <REDACTED:kind> for each match" do
      report = Redact.redact("email me at alice@example.com please")
      assert report.redacted =~ "<REDACTED:email>"
      refute report.redacted =~ "alice@example.com"
      assert length(report.matches) == 1
    end

    test "handles multiple matches without offset drift" do
      out =
        Redact.redact!(
          "a@b.com and 123-45-6789 and c@d.com",
          strategy: :placeholder
        )

      assert out == "<REDACTED:email> and <REDACTED:ssn> and <REDACTED:email>"
    end
  end

  describe "redact/2 — :mask" do
    test "replaces with * of the same length" do
      report = Redact.redact("call 555-123-4567 asap", strategy: :mask)
      assert String.contains?(report.redacted, "*")
      refute report.redacted =~ "555-123-4567"
    end
  end

  describe "redact/2 — :hash" do
    test "includes a short digest derived from the original" do
      out = Redact.redact!("from x@y.com", strategy: :hash)
      assert out =~ ~r/<REDACTED:email:[0-9a-f]{6}>/
    end
  end

  describe "redact/2 — :remove" do
    test "drops matches entirely" do
      out = Redact.redact!("mail x@y.com end", strategy: :remove)
      assert out == "mail  end"
    end
  end

  describe "opt-in / opt-out" do
    test "only: limits to selected kinds" do
      out = Redact.redact!("x@y.com and 123-45-6789", only: [:ssn])
      assert out =~ "x@y.com"
      assert out =~ "<REDACTED:ssn>"
    end

    test "except: excludes selected kinds" do
      out = Redact.redact!("x@y.com and 123-45-6789", except: [:ssn])
      assert out =~ "<REDACTED:email>"
      assert out =~ "123-45-6789"
    end
  end
end
