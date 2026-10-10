<script lang="ts">
  // 14 enterprise connectors as a card grid. Every card lists what gets
  // ingested + the principal model so the read is "what becomes
  // searchable + who is allowed to see it."

  type Conn = {
    name: string;
    glyph: string;
    kind: 'chat' | 'mail' | 'docs' | 'tickets' | 'meetings' | 'crm' | 'code' | 'storage';
    ingests: string;
    principal: string;
  };

  const connectors: Conn[] = [
    { name: 'Slack',     glyph: '#',  kind: 'chat',     ingests: 'channels · DMs · threads · files',           principal: 'channel + workspace ACL' },
    { name: 'Gmail',     glyph: '✉',  kind: 'mail',     ingests: 'inbox · labels · attachments',               principal: 'mailbox owner' },
    { name: 'Drive',     glyph: '◫',  kind: 'docs',     ingests: 'docs · sheets · slides · folders',           principal: 'file ACL + drive ACL' },
    { name: 'Notion',    glyph: '◇',  kind: 'docs',     ingests: 'pages · databases · blocks',                 principal: 'workspace + page ACL' },
    { name: 'Jira',      glyph: '◢',  kind: 'tickets',  ingests: 'issues · sprints · comments',                principal: 'project membership' },
    { name: 'Linear',    glyph: '◤',  kind: 'tickets',  ingests: 'issues · cycles · roadmaps',                 principal: 'team membership' },
    { name: 'GitHub',    glyph: '◐',  kind: 'code',     ingests: 'repos · PRs · issues · discussions',         principal: 'repo visibility + team' },
    { name: 'Zoom',      glyph: '◉',  kind: 'meetings', ingests: 'recordings → whisper transcripts',           principal: 'meeting host + invitees' },
    { name: 'Google Meet', glyph: '◎', kind: 'meetings', ingests: 'recordings · captions',                    principal: 'meeting invitees' },
    { name: 'Salesforce', glyph: '☁', kind: 'crm',      ingests: 'accounts · opportunities · activities',      principal: 'role hierarchy' },
    { name: 'HubSpot',   glyph: '◈',  kind: 'crm',      ingests: 'contacts · deals · sequences',               principal: 'team + assigned-owner' },
    { name: 'Confluence', glyph: '◇', kind: 'docs',     ingests: 'spaces · pages · attachments',               principal: 'space ACL' },
    { name: 'Dropbox',   glyph: '◰',  kind: 'storage',  ingests: 'folders · files · shares',                   principal: 'folder ACL' },
    { name: 'Outlook',   glyph: '✉',  kind: 'mail',     ingests: 'inbox · calendar · attachments',             principal: 'mailbox owner' }
  ];

  const kindColor: Record<Conn['kind'], string> = {
    chat:     '#7ea8ff',
    mail:     '#bb7eff',
    docs:     '#5fcfd4',
    tickets:  '#fc9e6c',
    meetings: '#f08acd',
    crm:      '#7be3a3',
    code:     '#9aa0a8',
    storage:  '#cfa45f'
  };
</script>

<svelte:head>
  <title>Connectors — Optimal Engine</title>
</svelte:head>

<section class="conn-hero">
  <div class="container">
    <span class="eyebrow">Mapping every digital signal in your company</span>
    <h1>Fourteen enterprise sources, one second brain.</h1>
    <p class="lede">
      A company doesn't think in one channel — it thinks across Slack, Gmail, Drive, Notion, Jira, GitHub, Zoom,
      Salesforce, Confluence, and more. Each connector ingests source-native objects, normalizes them into
      signal files inside the workspace, and tags every chunk with the principals allowed to see it. Permission
      scoping happens at retrieval — never bolted on after.
    </p>
  </div>
</section>

<section class="conn-grid-wrap">
  <div class="container">
    <div class="conn-grid">
      {#each connectors as c}
        <div class="conn-card">
          <div class="conn-card__head">
            <div class="conn-card__glyph" style="color: {kindColor[c.kind]}; background: color-mix(in srgb, {kindColor[c.kind]} 12%, transparent);">
              {c.glyph}
            </div>
            <div class="conn-card__title">
              <strong>{c.name}</strong>
              <span class="conn-card__kind" style="color: {kindColor[c.kind]};">{c.kind}</span>
            </div>
          </div>
          <dl class="conn-card__body">
            <dt>Ingests</dt>
            <dd>{c.ingests}</dd>
            <dt>Principal</dt>
            <dd>{c.principal}</dd>
          </dl>
        </div>
      {/each}
    </div>
  </div>
</section>

<style>
  .conn-hero { padding: 4rem 0 2rem; }
  .conn-hero h1 {
    font-size: clamp(2rem, 4vw, 3rem);
    margin: 0.6rem 0 1rem;
    letter-spacing: -0.01em;
  }
  .lede {
    color: var(--text-muted);
    font-size: 1.05rem;
    line-height: 1.6;
    max-width: 760px;
    margin: 0 0 1rem;
  }

  .conn-grid-wrap { padding: 2rem 0 4rem; }
  .conn-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: 0.75rem;
  }

  .conn-card {
    background: var(--bg-elevated);
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 1.1rem 1.2rem 1.2rem;
    transition: border-color 0.15s ease, transform 0.15s ease;
  }
  .conn-card:hover {
    border-color: var(--accent);
    transform: translateY(-1px);
  }
  .conn-card__head {
    display: flex;
    align-items: center;
    gap: 0.7rem;
    margin-bottom: 0.7rem;
  }
  .conn-card__glyph {
    width: 36px; height: 36px;
    border-radius: 10px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-size: 1.05rem;
    font-weight: 700;
  }
  .conn-card__title {
    display: flex;
    flex-direction: column;
    gap: 1px;
  }
  .conn-card__title strong {
    font-size: 0.98rem;
    color: var(--text);
  }
  .conn-card__kind {
    font-size: 0.7rem;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    font-weight: 600;
  }
  .conn-card__body {
    margin: 0;
    display: grid;
    grid-template-columns: max-content 1fr;
    gap: 0.35rem 0.75rem;
    font-size: 0.85rem;
  }
  .conn-card__body dt {
    color: var(--text-subtle);
    font-size: 0.7rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    font-weight: 600;
    align-self: center;
  }
  .conn-card__body dd {
    margin: 0;
    color: var(--text-muted);
  }
</style>
