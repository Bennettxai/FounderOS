defmodule OptimalEngine.Pipeline.Parser.HtmlTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Pipeline.Parser.Html

  test "extracts visible text and strips scripts/styles" do
    html = """
    <html>
      <head>
        <title>Test Page</title>
        <style>body { color: red }</style>
        <script>alert('x')</script>
      </head>
      <body>
        <h1>Welcome</h1>
        <p>Hello world.</p>
      </body>
    </html>
    """

    assert {:ok, doc} = Html.parse_text(html)
    assert doc.modality == :text
    refute String.contains?(doc.text, "body { color")
    refute String.contains?(doc.text, "alert('x')")
    assert String.contains?(doc.text, "Welcome")
    assert String.contains?(doc.text, "Hello world")
    assert doc.metadata.title == "Test Page"
  end

  test "extracts heading hierarchy" do
    html = "<h1>One</h1><h2>Two</h2><h3>Three</h3>"
    assert {:ok, doc} = Html.parse_text(html)
    headings = Enum.filter(doc.structure, &(&1.kind == :heading))
    assert length(headings) == 3
    assert Enum.map(headings, & &1.metadata.level) == [1, 2, 3]
  end
end
