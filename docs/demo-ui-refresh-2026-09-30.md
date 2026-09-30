# Demo UI refresh, September 30

The existing demo screens now use the current shared slab kit: rounded inset
cards, animated meters, step lines, dot matrices, volume summaries, and the
updated Daylight accent. The source UI baseline is BennettOS 53ca999.

FounderOS branding, Alex/Vantage/Launchpad fixtures, cohort surfaces, credential
resolution, and the demo database remain independent of the private OS. Org
markup is unchanged. Existing demo integrations continue to feed the new UI;
private client workspaces and newer private analytics integrations were not
imported.

Page and card cursor spotlights render nothing. Other interaction animations
remain enabled, including reduced-motion support.

Privacy guards now prevent public demo mode from scanning local WhatsApp chats,
installed skills, model usage transcripts, or G-Brain CLI/store data. Regression
tests cover those boundaries and scan application source for private names and
hosts. No credentials or personal databases were copied.

Validation: full Vitest suite, TypeScript, and HTTP checks of 18 page/API routes.
Browser preview at http://localhost:4137 uses a fresh database under /tmp and
explicit DEMO_GATE=1. Home renders the new slab, generic operator name, and zero
spotlight elements. Comms, Skills, and Doctor responses were checked again after
the host-read guards were added.

Publication was explicitly authorized with "push" on September 30. Remote main
changes through ad46d77 were merged cleanly before publication. The merged
result passes all 3,464 tests across 313 files and TypeScript checking.
