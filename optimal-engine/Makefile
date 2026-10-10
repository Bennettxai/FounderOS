# Optimal Engine - Make targets
#
# make install        install deps + compile
# make bootstrap      migrate + ingest sample-workspace/
# make dev            boot the HTTP engine with a complete local runtime
# make repl           boot iex for direct engine debugging
# make test           full test suite
# make reality        run the reality-check probes
# make docs-check     public docs/repo hygiene checks
# make clean          wipe _build/ and the dev SQLite

.PHONY: install bootstrap dev repl test reality docs-check clean seed help

help:
	@echo "Optimal Engine - make targets"
	@echo ""
	@echo "  make install      - mix deps.get + mix compile"
	@echo "  make bootstrap    - compile, migrate, ingest sample-workspace/"
	@echo "  make dev          - run the HTTP engine with a local connector key"
	@echo "  make repl         - iex -S mix for direct engine debugging"
	@echo "  make test         - full test suite"
	@echo "  make reality      - mix optimal.reality_check --hard"
	@echo "  make docs-check   - public audit + whitespace check"
	@echo "  make clean        - wipe _build/ and the dev SQLite"
	@echo ""
	@echo "  CLI:          bin/optimal doctor"
	@echo "  Quick start:  bin/optimal install && bin/optimal bootstrap && bin/optimal dev"

install:
	mix deps.get
	mix compile

bootstrap: install
	mix optimal.bootstrap

seed:
	mix optimal.bootstrap

dev:
	scripts/run-engine.sh

repl:
	iex -S mix

test:
	mix test

reality:
	mix optimal.reality_check --hard

docs-check:
	scripts/public-audit.sh
	git diff --check

clean:
	rm -rf _build/ .optimal/index.db*
