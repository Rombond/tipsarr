const H = storage.H, C = storage.C, S = storage.S;
const F = S.F = {};
F.section = (b, T, x, y, title, note) => { H.text(b, title, x, y, { size: 28, weight: 700, color: T.fg, ls: -0.3 }); if (note) H.text(b, note, x, y + 38, { size: 14, color: T.mutedFg, w: 1100 }); H.rect(b, 'rule', x, y + (note ? 70 : 44), 1340, 1, T.border); };
F.build = (T, x, y, part) => {
  const b = H.board(null, `Foundations · ${part} · ${T.name}`, x, y, 1440, part === 'colors' ? 1500 : part === 'type' ? 1100 : 1900, { fill: T.bg, r: 0 });
  if (part === 'colors') {
    H.text(b, 'Tipsarr · foundations', 50, 40, { size: 40, weight: 700, color: T.fg, ls: -0.5 }); H.text(b, `Neutral theme (${T.name}) mirrors the web: black / white / grey, with the status colours carrying the colour. Inter stands in for SF Pro (iOS) and Roboto (Android).`, 50, 92, { size: 16, color: T.mutedFg, w: 1100 });
    F.section(b, T, 50, 160, 'Neutral palette', 'Token sets: core + light / dark (theme "mode"). Values here are the real ones used by the screens.');
    const sw = [['bg', T.bg], ['fg', T.fg], ['card', T.card], ['muted', T.muted], ['mutedFg', T.mutedFg], ['border', T.border], ['primary', T.primary], ['primaryFg', T.primaryFg], ['destructive', T.destructive]];
    sw.forEach(([n, c], i) => { const col = i % 5, row = Math.floor(i / 5); const px = 50 + col * 270, py = 260 + row * 150; H.rect(b, 'swatch ' + n, px, py, 250, 84, c, { r: 14, stroke: [T.isDark ? '#FFFFFF' : '#000000', 1, 0.15] }); H.text(b, `color.${n}`, px, py + 94, { size: 14, weight: 600, color: T.fg }); H.text(b, c.toUpperCase(), px, py + 114, { size: 12, color: T.mutedFg }); });
    F.section(b, T, 50, 600, 'Status colours', 'One colour per state, same on web and mobile. Solid (posters) and soft (rows, detail).');
    Object.entries(C.status).forEach(([k, s], i) => { const col = i % 4, row = Math.floor(i / 4); const px = 50 + col * 340, py = 700 + row * 130; H.rect(b, 'swatch ' + k, px, py, 60, 60, s.c, { r: 14 }); H.icon(b, s.icon, px + 16, py + 16, 28, '#FFFFFF', 2.2); H.text(b, s.label, px + 76, py + 2, { size: 17, weight: 600, color: T.fg }); H.text(b, s.c, px + 76, py + 26, { size: 12, color: T.mutedFg }); C.badge(b, T, px + 76, py + 48, k, {}); C.badge(b, T, px + 200, py + 48, k, { solid: true }); });
    F.section(b, T, 50, 1000, 'Spacing & radius', 'Spacing 4 · 8 · 12 · 16 · 20 · 24 · 32 · 48. Radius 8 · 12 · 16 · 20 · 28 (sheets) · full.');
    [4, 8, 12, 16, 20, 24, 32, 48].forEach((v, i) => { H.rect(b, 'space ' + v, 50 + i * 120, 1100, v, 48, T.fg, { r: 2 }); H.text(b, String(v), 50 + i * 120, 1160, { size: 13, color: T.mutedFg }); });
    [8, 12, 16, 20, 28, 40].forEach((v, i) => { H.rect(b, 'radius ' + v, 50 + i * 170, 1220, 120, 120, T.muted, { r: v, stroke: [T.border, 1] }); H.text(b, `radius ${v}`, 50 + i * 170, 1350, { size: 13, color: T.mutedFg }); });
    H.text(b, 'Poster ratio 2 : 3  ·  tab bar floating pill  ·  sheet top radius 28  ·  touch target ≥ 44 pt', 50, 1420, { size: 14, color: T.mutedFg });
  }
  if (part === 'type') {
    F.section(b, T, 50, 40, 'Type scale', 'Mobile uses the system font (SF Pro / Roboto). Sizes follow iOS text styles; web equivalents in brackets.');
    [['Large title', 34, 700], ['Title 1', 28, 700], ['Title 2', 22, 700], ['Title 3', 20, 700], ['Headline', 17, 600], ['Body', 17, 400], ['Callout', 16, 400], ['Subhead', 15, 400], ['Footnote', 13, 400], ['Caption', 12, 500]].forEach(([n, s, w], i) => { const yy = 130 + i * 92; H.text(b, n, 50, yy + 8, { size: 13, weight: 600, color: T.mutedFg, name: 'style name' }); H.text(b, `${s} pt  ·  ${w}`, 50, yy + 28, { size: 12, color: T.mutedFg }); H.text(b, 'Dune: Part Two — The quick brown fox', 260, yy, { size: s, weight: w, color: T.fg, name: 'sample ' + n }); });
  }
  if (part === 'components') {
    F.section(b, T, 50, 40, 'Buttons', 'Primary · secondary · outline · ghost · destructive · success; with icon, disabled.');
    [['primary', 'Request', 'plus'], ['secondary', 'Cancel request', null], ['outline', 'Details', 'info'], ['ghost', 'Not now', null], ['destructive', 'Decline', 'x'], ['success', 'Open in Jellyfin', 'play']].forEach(([v, l, ic], i) => C.btn(b, T, 50 + (i % 3) * 420, 130 + Math.floor(i / 3) * 70, 390, l, { v, icon: ic, h: 52 }));
    C.btn(b, T, 50, 270, 390, 'Disabled', { v: 'primary', disabled: true, h: 52 });
    F.section(b, T, 50, 360, 'Chips, segmented, badges');
    let cx = 50; [['Trending', true, 'trending-up'], ['Upcoming', false, 'calendar'], ['Movies', false, 'film']].forEach(([l, a, ic]) => { const c = C.chip(b, T, cx, 440, l, { active: a, icon: ic }); cx += c.width + 10; }); C.segmented(b, T, 470, 440, 420, ['All', 'Movies', 'TV', 'People'], 1);
    Object.keys(C.status).forEach((k, i) => { C.badge(b, T, 50 + i * 150, 500, k, {}); C.badge(b, T, 50 + i * 150, 534, k, { solid: true }); });
    F.section(b, T, 50, 600, 'Fields, banners', 'Focus ring = 1.5 pt foreground. Error = destructive stroke + message.');
    C.field(b, T, 50, 680, 400, 'Username', { value: 'alice', icon: 'user' }); C.field(b, T, 480, 680, 400, 'Password', { value: '••••••', secure: true, icon: 'key-round', focus: true }); C.field(b, T, 910, 680, 400, 'Server', { value: 'tipsarr.example', icon: 'globe', error: true, msg: 'Can’t reach this server.' });
    C.banner(b, T, 50, 810, 400, 'error', 'Declined by an admin: not available.'); C.banner(b, T, 480, 810, 400, 'warn', 'The download failed. An admin can retry it.'); C.banner(b, T, 910, 810, 400, 'info', 'Your session expires in 2 days.');
    F.section(b, T, 50, 930, 'Posters', 'Placeholder gradients stand in for TMDB artwork. Status icon top-right, optional rating top-left, watched eye.');
    [0, 1, 2, 3, 4, 5].forEach((n, i) => { const p = C.poster(b, T, 50 + i * 150, 1010, 130, n, { rating: i === 2 }); if (i === 5) C.watchedMark(p, T, 130 - 28, 6); });
    F.section(b, T, 50, 1260, 'Lists & navigation', 'Grouped list rows, tab bar (floating pill), navigation bar, section header.');
    C.group(b, T, 50, 1340, 400, [{ title: 'Language', icon: 'languages', value: 'English' }, { title: 'Lock with Face ID', icon: 'scan-face', toggle: true }, { title: 'Sign out', icon: 'log-out', danger: true, chev: false }]);
    const tbHolder = H.board(b, 'tab bar demo', 480, 1340, 393, 100, { fill: T.bg, r: 16, stroke: [T.border, 1], clip: true }); const tb = C.tabBar(tbHolder, T, 2); tb.y = tbHolder.y + 18;
    C.sectionHeader(b, T, 910, 1340, 360, 'Trending now'); H.text(b, 'Large title', 910, 1390, { size: 34, weight: 700, color: T.fg });
    F.section(b, T, 50, 1500, 'Icons (Lucide, ISC)', 'Same set as the web (@lucide/svelte); stroke 2 at 24 pt, 1.8 in lists.');
    Object.keys(storage.icons).forEach((n, i) => { const col = i % 16, row = Math.floor(i / 16); const px = 50 + col * 84, py = 1590 + row * 84; H.icon(b, n, px + 20, py, 28, T.fg, 1.9); H.text(b, n, px - 6, py + 36, { size: 9, color: T.mutedFg, w: 80, align: 'center' }); });
  }
  return b;
};
S.foundList = [['colors', 0], ['type', 1], ['components', 2]];
