defmodule OptimalEngine.MixProject do
  use Mix.Project

  @version "0.3.0"

  def project do
    [
      app: :optimal_engine,
      version: @version,
      elixir: "~> 1.17",
      start_permanent: Mix.env() == :prod,
      deps: deps(),
      aliases: aliases(),
      elixirc_paths: elixirc_paths(Mix.env()),
      dialyzer: [plt_add_apps: [:mix]],
      description: description(),
      package: package(),
      escript: escript(),
      releases: releases()
    ]
  end

  # Local source-checkout CLI wrapper. It routes `./optimal <cmd>` to
  # `mix optimal.<cmd>` so native dependencies load from the normal Mix build.
  # It is not the production packaging format.
  defp escript do
    [
      main_module: OptimalEngine.CLI,
      name: "optimal",
      app: nil
    ]
  end

  # `MIX_ENV=prod mix release optimal` produces a supervised OTP runtime in
  # _build/prod/rel/optimal/. Use that shape for API/server deployment.
  defp releases do
    [
      optimal: [
        include_executables_for: [:unix],
        applications: [optimal_engine: :permanent]
      ]
    ]
  end

  def application do
    [
      extra_applications: [:logger, :crypto, :inets, :ssl, :mnesia],
      mod: {OptimalEngine.Application, []}
    ]
  end

  defp elixirc_paths(:test), do: ["lib", "test/support"]
  defp elixirc_paths(_), do: ["lib"]

  defp deps do
    [
      # Storage & serialization
      {:exqlite, "~> 0.27"},
      {:jason, "~> 1.4"},
      {:yaml_elixir, "~> 2.11"},

      # Optional knowledge graph substrate
      {:rdf, "~> 2.0"},
      {:rocksdb, "~> 3.1"},

      # Signal and option validation
      {:nimble_options, "~> 1.1"},
      {:uuid, "~> 1.1"},

      # HTTP API
      {:plug, "~> 1.15"},
      {:plug_cowboy, "~> 2.7"},

      # Parser backends
      {:nimble_csv, "~> 1.2"},
      {:floki, "~> 0.36"},

      # Observability
      {:telemetry, "~> 1.2"},

      # Auth
      {:bcrypt_elixir, "~> 3.1"},

      # Dev / test
      {:credo, "~> 1.7", only: [:dev, :test], runtime: false},
      {:dialyxir, "~> 1.4", only: [:dev], runtime: false}
    ]
  end

  defp aliases do
    [
      "optimal.index": ["run -e 'Mix.Tasks.Optimal.Index.run([])'"],
      "optimal.search": ["run -e 'Mix.Tasks.Optimal.Search.run(System.argv())'"],
      # Wipe the test SQLite store before every run so schema drift from
      # an older migration set can't leak into the current test process.
      test: [
        "cmd rm -f /tmp/optimal_engine_test_0.db /tmp/optimal_engine_test_0.db-wal /tmp/optimal_engine_test_0.db-shm",
        "test"
      ]
    ]
  end

  defp description do
    "Self-hosted second brain and operating engine for human and AI workspaces."
  end

  defp package do
    [
      licenses: ["MIT"],
      links: %{"GitHub" => "https://github.com/Miosa-osa/OptimalEngine"},
      files:
        ~w(bin lib priv config docs mix.exs README.md LICENSE AGENTS.md CLAUDE.md BOOT.md SYSTEM.md OPTIONS.md MISSION.md RESOURCES.md NOTES.md OPINIONS.md)
    ]
  end
end
