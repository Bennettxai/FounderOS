import Link from 'next/link';
import { getDb } from '@/lib/data';
import { SkillsGrid, type SkillCard } from '@/components/SkillsGrid';
import { readPluginSkills, readUserSkills } from '@/lib/skills-catalog';
import { skillsVolume } from '@/lib/skills-volume';
import { Slab, SlabTitle, SlabCard, BigStat, Chip, MeterStack, InsightCard, PILL } from '@/components/slab';
import { DotMatrix } from '@/components/slab-charts';

export const dynamic = 'force-dynamic';

const truncate = (t: string, n = 110) => (t.length > n ? `${t.slice(0, n).replace(/\s+\S*$/, '')}…` : t);

/**
 * 2026-09-24: the Brand Deals slab. Skills carry no dates, so the hero shows
 * the catalog's shape (skills per source group) rather than a timeline it
 * would have to invent. Every number comes from lib/skills-volume over the
 * same rows the card wall renders; the wall keeps its filter, reader and
 * download and takes the stagger at i={6}.
 */
export default function SkillsPage() {
  // All three catalogs on one wall: the real Claude Code skills read live
  // from disk (user-scope ~/.claude/skills plus every installed plugin's
  // skills; SKILL.md loads on demand via /api/skills/[slug]) alongside the
  // operator skill cards that have always lived in this section.
  const real = [...readUserSkills(), ...readPluginSkills()];
  const realCards: SkillCard[] = real.map((s) => ({
    id: s.slug,
    name: s.name,
    group: s.group,
    kind: 'claude' as const,
    description: truncate(s.description),
    meta: s.path,
    filePath: s.path,
  }));

  const db = getDb();
  const agentNames = Object.fromEntries(db.agents.all().map((a) => [a.id, a.name]));
  const operator = db.skills.all();
  const operatorCards: SkillCard[] = operator.map((s) => ({
    id: s.id,
    name: s.name,
    group: `Operator · ${s.category}`,
    kind: 'operator' as const,
    description: truncate(s.description),
    meta: s.ownerAgentId ? (agentNames[s.ownerAgentId] ?? s.ownerAgentId) : 'unassigned',
    filePath: `skills/${s.id}/SKILL.md`,
    status: s.status,
    markdown: s.markdown,
  }));

  const cards = [...realCards, ...operatorCards];
  const sourceNote =
    real.length > 0
      ? `${real.length} skills live from ~/.claude (user + plugins) + ${operatorCards.length} operator skills · open any card to read or download its SKILL.md.`
      : `${operatorCards.length} operator skills (no ~/.claude/skills on this machine) · open any card to read or download its SKILL.md.`;

  const v = skillsVolume({ claude: real, operator, agentNames });
  const topCategory = v.categories[0];

  return (
    <Slab>
      <SlabTitle
        eyebrow="capability library"
        title="Skills"
        meta={`${v.counts.claude} Claude Code skills on disk · ${v.counts.operator} operator skills · ${v.groupCount} groups`}
        right={
          <>
            <Chip tone={v.insight.value > 0 ? 'warn' : 'ok'}>
              {v.counts.live} live · {v.insight.value} drafts
            </Chip>
            <Link href="/agents" className={PILL}>
              Agents
            </Link>
          </>
        }
      />

      {/* Hero row, Brand Deals' shape: the catalog's shape + the volume card */}
      <div className="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
        <SlabCard i={1} title="Skill Library" sub="skills per source group">
          {/* stacked: group names are long column labels */}
          <div className="flex flex-col gap-6 px-6 pb-6 pt-3">
            <BigStat
              size={30}
              value={v.groupCount}
              unit="groups"
              chips={[{ tone: 'accent', text: `${v.counts.plugin} from plugins` }, { text: `${v.counts.user} user` }]}
              caption={v.groups.length > 0 ? `largest · ${v.groups[0].label} with ${v.groups[0].count}` : 'no skills on this machine yet'}
            />
            {v.groups.length > 0 && <DotMatrix cols={v.groups} hue="var(--send-activity)" />}
          </div>
        </SlabCard>

        <SlabCard i={2} title="Skill Volume" className="flex flex-col">
          <div className="flex flex-1 flex-col px-6 pb-6 pt-3">
            <BigStat value={v.headline} chips={v.chips} caption={v.caption} />
            <MeterStack meters={v.meters} foot={v.foot} empty="no skills yet" />
          </div>
        </SlabCard>
      </div>

      {/* Second row: operator categories, owners, THE gradient card */}
      <div className="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
        <SlabCard i={3} title="Categories" sub="operator skills">
          <div className="flex flex-col gap-5 px-6 pb-6 pt-3">
            <BigStat
              size={30}
              value={v.categories.length}
              caption={topCategory ? `most in ${topCategory.label} · ${topCategory.count}` : 'no operator skills yet'}
            />
            {v.categories.length > 0 && <DotMatrix cols={v.categories} hue="var(--ramp-1)" />}
          </div>
        </SlabCard>

        <SlabCard i={4} title="Owners" sub="agents wielding them">
          <div className="flex flex-col gap-5 px-6 pb-6 pt-3">
            <BigStat
              size={30}
              value={v.owners.length}
              chips={v.unassigned > 0 ? [{ tone: 'warn', text: `${v.unassigned} unassigned` }] : []}
              caption={v.owners.length > 0 ? `${v.owners[0].label} holds the most · ${v.owners[0].count}` : 'no operator skill has an owner yet'}
            />
            {v.owners.length > 0 && <DotMatrix cols={v.owners} hue="var(--ramp-4)" />}
          </div>
        </SlabCard>

        <InsightCard i={5} badge="Drafts" value={v.insight.value} headline={v.insight.headline} body={v.insight.body} frac={v.insight.frac} />
      </div>

      <SkillsGrid i={6} cards={cards} sourceNote={sourceNote} />
    </Slab>
  );
}
