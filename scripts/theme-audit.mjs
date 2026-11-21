// Theme audit: screenshots every page in every theme and scans computed styles for colour leaks.
// Start Chrome first:  "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" --headless=new --remote-debugging-port=9333 --user-data-dir=/tmp/glados-audit --window-size=1440,900 about:blank &
// Then:  S=/tmp/glados-audit BASE=http://127.0.0.1:4300 node scripts/theme-audit.mjs [themes,comma|all] [routes,comma|all] [outdir]
// Themes default to every id in lib/theme.ts; routes to scripts/audit-routes.mjs (or ROUTES="/ /inbox").
// Reports literal greens (not the value of any token), token greens and text shadows. Exits 1 when
// Red, Frost or Grey Frost show a green of either kind outside the sanctioned routes; in Console and
// White (green signal) greens are reported, not failed: the detector cannot trace rgb-triplet tokens
// such as --live-badge-rgb or the engine paper hues back to a token. Non-GET /api requests are blocked.
import fs from 'node:fs';
import { THEME_IDS, ZERO_GREEN_THEMES, cdp, routesFromEnv } from './audit-routes.mjs';
import { setThemeExpr } from './theme-ground.mjs';
const [,, themesArg = 'all', routesArg = 'all', outArg] = process.argv;
const BASE = process.env.BASE ?? 'http://127.0.0.1:4302';
const OUT = outArg ?? `${process.env.S}/audit/out`; fs.mkdirSync(OUT, { recursive: true });
const routes = routesFromEnv(routesArg);
const themes = themesArg === 'all' ? THEME_IDS : themesArg.split(',');
const { send, ev, close } = await cdp();
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
await send('Emulation.setDeviceMetricsOverride', { width: Number(process.env.WIDTH ?? 1440), height: 900, deviceScaleFactor: 1, mobile: false });
// the leak scanner: every visible element's text colour, background, border, box-shadow, text-shadow, svg fill/stroke
const SCAN = `(() => {
  const toHsl = (r,g,b) => { r/=255; g/=255; b/=255; const mx=Math.max(r,g,b), mn=Math.min(r,g,b); let h=0,s=0; const l=(mx+mn)/2; const d=mx-mn; if(d){ s = l>0.5 ? d/(2-mx-mn) : d/(mx+mn); switch(mx){ case r: h=(g-b)/d+(g<b?6:0); break; case g: h=(b-r)/d+2; break; default: h=(r-g)/d+4; } h*=60; } return [h,s,l]; };
  const parse = (s) => { const m = s && s.match(/rgba?\\(([^)]+)\\)/); if(!m) return null; const [r,g,b,a='1'] = m[1].split(',').map(x=>parseFloat(x)); return a < 0.08 ? null : [r,g,b,a]; };
  const theme = document.documentElement.getAttribute('data-theme');
  // every custom property value currently in effect on :root → the set of rgb triples a token can legitimately produce
  const allowed = new Set(); const names = new Set();
  for (const sh of document.styleSheets) { let rules; try { rules = sh.cssRules; } catch { continue; } for (const r of rules) { if (!r.style) continue; for (const n of r.style) if (n.startsWith('--')) names.add(n); } }
  const rootCs = getComputedStyle(document.documentElement);
  const probe = document.createElement('div'); probe.style.position='fixed'; probe.style.left='-9999px'; document.body.appendChild(probe);
  for (const n of names) { const raw = rootCs.getPropertyValue(n).trim(); if (!raw) continue; probe.style.color = ''; probe.style.color = raw; const c = parse(getComputedStyle(probe).color); if (c) allowed.add(c.slice(0,3).join(',')); const trip = raw.match(/^\d+\s+\d+\s+\d+$/); if (trip) allowed.add(raw.split(/\s+/).join(',')); for (const m of raw.match(/rgba?\([^)]+\)|#[0-9a-fA-F]{3,8}\b/g) || []) { probe.style.color=''; probe.style.color = m; const c2 = parse(getComputedStyle(probe).color); if (c2) allowed.add(c2.slice(0,3).join(',')); } }
  probe.remove();
  const findings = []; const seen = new Set();
  const isGreen = (h,s,l) => h >= 95 && h <= 175 && s >= 0.35 && l >= 0.18 && l <= 0.92;
  const tag = (el) => { const c = (el.className && typeof el.className === 'string') ? el.className.split(/\\s+/).slice(0,3).join('.') : ''; return el.tagName.toLowerCase() + (el.id ? '#'+el.id : '') + (c ? '.'+c : ''); };
  for (const el of document.querySelectorAll('body *')) {
    const r = el.getBoundingClientRect(); if (r.width < 2 || r.height < 2 || r.bottom < 0 || r.top > 4000) continue;
    const cs = getComputedStyle(el); if (cs.visibility === 'hidden' || cs.opacity === '0') continue;
    const props = [['color', cs.color], ['background', cs.backgroundColor], ['border', cs.borderTopColor], ['fill', cs.fill], ['stroke', cs.stroke]];
    for (const [k, v] of props) { const p = parse(v); if (!p) continue; if (k==='border' && cs.borderTopWidth === '0px') continue; if (k==='fill' && el.tagName.toLowerCase() !== 'path' && el.tagName.toLowerCase() !== 'circle' && el.tagName.toLowerCase() !== 'rect' && el.tagName.toLowerCase() !== 'svg') continue; const [h,s,l] = toHsl(p[0],p[1],p[2]); if (isGreen(h,s,l)) { const key = k+'|'+v+'|'+tag(el); if (seen.has(key)) continue; seen.add(key); findings.push({ kind: allowed.has(p.slice(0,3).join(',')) ? 'green-token' : 'green', prop: k, value: v, el: tag(el), text: (el.textContent||'').trim().slice(0,40) }); } }
    for (const [k, v] of [['boxShadow', cs.boxShadow], ['backgroundImage', cs.backgroundImage]]) { if (!v || v === 'none') continue; const ms = v.match(/rgba?\\([^)]+\\)/g) || []; for (const m of ms) { const p = parse(m); if (!p) continue; const [h,s,l] = toHsl(p[0],p[1],p[2]); if (isGreen(h,s,l)) { const key = k+'|'+m+'|'+tag(el); if (seen.has(key)) continue; seen.add(key); findings.push({ kind: allowed.has(p.slice(0,3).join(',')) ? 'green-token' : 'green', prop: k, value: m, el: tag(el) }); } } }
    if (cs.textShadow && cs.textShadow !== 'none') { const key = 'ts|'+tag(el); if (!seen.has(key)) { seen.add(key); findings.push({ kind: 'text-shadow', value: cs.textShadow.slice(0,60), el: tag(el), text: (el.textContent||'').trim().slice(0,30) }); } }
  }
  return { theme, url: location.pathname, findings, title: document.title, bodyBg: getComputedStyle(document.body).backgroundColor };
})()`;
const report = [];
for (const theme of themes) {
  await send('Page.addScriptToEvaluateOnNewDocument', { source: `try{localStorage.setItem('glados-theme',${JSON.stringify(theme)})}catch(e){}` });
  for (const route of routes) {
    const name = `${theme}--${route === '/' ? 'home' : route.replace(/^\//, '').replace(/[/?=&]/g, '_')}`;
    try {
      await send('Page.navigate', { url: BASE + route }); await sleep(Number(process.env.WAIT ?? 4500));
      await ev(setThemeExpr(theme)); await sleep(400); // data-theme and the data-ground the glass light reads
      const r = await send('Page.captureScreenshot', { format: 'png' }); fs.writeFileSync(`${OUT}/${name}.png`, Buffer.from(r.data, 'base64'));
      const scan = await ev(SCAN);
      const greens = scan.findings.filter((f) => f.kind === 'green'); const tokenGreens = scan.findings.filter((f) => f.kind === 'green-token'); const shadows = scan.findings.filter((f) => f.kind === 'text-shadow');
      report.push({ theme, route, literalGreens: greens.length, tokenGreens: tokenGreens.length, shadows: shadows.length, bodyBg: scan.bodyBg, sample: greens.slice(0, 8), tokenSample: tokenGreens.slice(0, 6), shadowSample: shadows.slice(0, 4) });
      console.log(`${name}: ${greens.length} literal-green, ${tokenGreens.length} token-green, ${shadows.length} text-shadow, bg ${scan.bodyBg}`);
    } catch (e) { console.log(`${name}: ERROR ${e.message.slice(0, 80)}`); report.push({ theme, route, error: e.message }); }
  }
}
fs.writeFileSync(`${OUT}/report-${themes.join('-')}.json`, JSON.stringify(report, null, 1));
close();
// Sanctioned (docs/THEMES.md): Setup's theme picker draws every theme's swatch (S06 edge 14);
// Integrations shows third-party brand marks in their own colours. Known and owned elsewhere:
// none (/personas, the last one, was removed by SPEC S12).
const SANCTIONED = new Set(['/setup', '/integrations', '/integrations/whatsapp']);
const KNOWN = new Set([]);
const fails = report.filter((r) => !SANCTIONED.has(r.route) && !KNOWN.has(r.route) && (r.error || (ZERO_GREEN_THEMES.includes(r.theme) && (r.literalGreens > 0 || r.tokenGreens > 0))));
const known = report.filter((r) => KNOWN.has(r.route) && (r.literalGreens > 0 || r.tokenGreens > 0));
if (known.length) console.log(`known, owned elsewhere: ${known.map((r) => `${r.theme} ${r.route}`).join(', ')}`);
for (const t of themes) {
  const rows = report.filter((r) => r.theme === t);
  const sum = (k) => rows.reduce((n, r) => n + (r[k] ?? 0), 0);
  console.log(`${t}: ${rows.length} routes, ${sum('literalGreens')} literal greens, ${sum('tokenGreens')} token greens, ${sum('shadows')} text shadows`);
}
if (fails.length) { console.log(`FAIL ${fails.map((r) => `${r.theme} ${r.route}`).join(', ')}`); process.exitCode = 1; } else console.log('PASS');
