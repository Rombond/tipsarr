const H = storage.H, C = storage.C, S = storage.S;
const P = S.P = {};
P.board = (T, name, x, y, w, h) => H.board(null, name, x, y, w, h, { fill: T.bg, r: 0 });
P.statusBar = (b, T, w) => { H.text(b, '9:41', 24, 10, { size: 14, weight: 600, color: T.fg, name: 'time' }); H.text(b, 'Tue Oct 8', 80, 10, { size: 14, weight: 600, color: T.fg }); const bx = w - 24 - 25; H.rect(b, 'battery', bx, 12, 25, 12, null, { r: 4, stroke: [T.fg, 1, 0.4] }); H.rect(b, 'battery level', bx + 2, 14, 21, 8, T.fg, { r: 2.5 }); };
P.sidebar = (b, T, active, o = {}) => {
  const w = o.collapsed ? 84 : 288; const sb = H.board(b, 'sidebar', 0, 0, w, b.height, { fill: T.isDark ? '#101010' : '#F7F7F8', clip: true, stroke: null }); H.rect(sb, 'edge', w - 1, 0, 1, b.height, T.border);
  C.appIcon(sb, o.collapsed ? 24 : 20, 44, 36, T.isDark ? 'Dark' : 'Light'); if (!o.collapsed) H.text(sb, 'Tipsarr', 64, 48, { size: 22, weight: 700, color: T.fg });
  [['compass', 'Discover'], ['search', 'Search'], ['list-checks', 'Requests'], ['library', 'Library'], ['user', 'Profile']].forEach(([ic, l], i) => { const on = i === active; const yy = 112 + i * 52; if (on) H.rect(sb, 'active', o.collapsed ? 14 : 12, yy, w - (o.collapsed ? 28 : 24), 44, T.muted, { r: 12 }); H.icon(sb, ic, o.collapsed ? 30 : 28, yy + 11, 22, on ? T.fg : T.mutedFg, on ? 2.2 : 1.8); if (!o.collapsed) H.text(sb, l, 64, yy + 12, { size: 17, weight: on ? 600 : 500, color: on ? T.fg : T.mutedFg }); });
  if (!o.collapsed) { C.avatarSm(sb, T, 20, b.height - 80, 'Alice', 3, 36); H.text(sb, 'Alice', 64, b.height - 80, { size: 15, weight: 600, color: T.fg }); H.text(sb, 'tipsarr.brebond.com', 64, b.height - 60, { size: 12, color: T.mutedFg }); }
  P.statusBar(b, T, b.width);
  return w;
};
P.rail = (b, T, x, y, w, title, idxs, pw = 140) => { H.text(b, title, x, y, { size: 22, weight: 700, color: T.fg }); H.text(b, 'See all', x, y + 4, { size: 15, weight: 500, color: T.mutedFg, w, align: 'right' }); idxs.forEach((n, i) => { const px = x + i * (pw + 14); if (px + pw <= x + w + 4) { C.poster(b, T, px, y + 40, pw, n, { badge: true }); } }); };
P.discover = (T, x, y) => {
  const b = P.board(T, `iPad · Discover · ${T.name}`, x, y, 1194, 834); const sw = P.sidebar(b, T, 0);
  const cx = sw + 32, cw = 1194 - sw - 64; const pal = C.pal[0];
  H.board(b, 'hero', sw, 0, 1194 - sw, 380, { fill: [H.grad([[pal[0], 0], [pal[1], 0.7], [T.bg, 1]])], clip: true });
  H.text(b, 'Dune: Part Two', cx, 190, { size: 44, weight: 700, color: '#FFFFFF', ls: -0.6 }); H.text(b, '2024  ·  Movie  ·  2h 46m  ·  Sci-Fi', cx, 248, { size: 16, weight: 500, color: '#D4D4D8' });
  H.text(b, 'Paul Atreides unites with Chani and the Fremen while seeking revenge against the conspirators who destroyed his family.', cx, 280, { size: 16, color: '#E5E5E5', w: 520, lh: 1.4 }); C.btn(b, T, cx, 332, 160, 'Request', { icon: 'plus', h: 46 }); C.btn(b, T, cx + 172, 332, 130, 'Details', { v: 'secondary', h: 46, icon: 'info' });
  P.rail(b, T, cx, 408, cw, 'Trending now', [1, 2, 3, 4, 5, 6], 132); P.rail(b, T, cx, 676, cw, 'Upcoming', [7, 8, 9, 10, 11, 0], 132);
  return b;
};
P.master = (T, x, y) => {
  const b = P.board(T, `iPad · Search + detail · ${T.name}`, x, y, 1194, 834); const sw = P.sidebar(b, T, 1, { collapsed: true });
  const lw = 380; H.rect(b, 'pane edge', sw + lw, 0, 1, 834, T.border); C.search(b, T, sw + 16, 50, lw - 32, 'Movies, shows, people…', { value: 'dune' });
  [0, 9, 3, 5, 11].forEach((n, k) => { const yy = 120 + k * 122; if (k === 0) H.rect(b, 'selected', sw + 8, yy - 8, lw - 16, 116, T.muted, { r: 14 }); C.resultRow(b, T, sw + 20, yy, lw - 40, n, { overview: 'A short synopsis wraps here.' }); });
  const dx = sw + lw; const dw = 1194 - dx; const pal = C.pal[0];
  H.board(b, 'backdrop', dx + 1, 0, dw - 1, 330, { fill: [H.grad([[pal[0], 0], [pal[1], 0.82], [T.bg, 1]])], clip: true });
  C.poster(b, T, dx + 32, 150, 150, 0, { badge: false, noTitle: true, r: 14 }); H.text(b, 'Dune: Part Two', dx + 206, 190, { size: 34, weight: 700, color: '#FFFFFF' }); H.text(b, '2024  ·  2h 46m  ·  PG-13', dx + 206, 236, { size: 15, color: '#D4D4D8' }); C.badge(b, T, dx + 206, 266, 'available', {});
  C.btn(b, T, dx + 32, 348, 190, 'Open in Jellyfin', { icon: 'play', v: 'success', h: 48 }); const mb = H.board(b, 'menu', dx + 234, 348, 48, 48, { fill: T.muted, r: 24, clip: false }); H.icon(mb, 'ellipsis', 12, 12, 24, T.fg, 2);
  H.text(b, 'Overview', dx + 32, 424, { size: 22, weight: 700, color: T.fg }); H.text(b, 'Paul Atreides unites with Chani and the Fremen while seeking revenge against the conspirators who destroyed his family. Facing a choice between the love of his life and the fate of the known universe, he endeavors to prevent a terrible future only he can foresee.', dx + 32, 460, { size: 16, color: T.fg, w: dw - 64, lh: 1.45, op: 0.85 });
  H.text(b, 'Cast', dx + 32, 580, { size: 22, weight: 700, color: T.fg }); [['Timothée Chalamet', 'Paul'], ['Zendaya', 'Chani'], ['Rebecca Ferguson', 'Jessica'], ['Josh Brolin', 'Gurney'], ['Austin Butler', 'Feyd']].forEach(([n, r], j) => C.avatarCol(b, T, dx + 24 + j * 92, 622, n, r, j + 1));
  return b;
};
P.requests = (T, x, y) => {
  const b = P.board(T, `iPad · Requests · ${T.name}`, x, y, 1194, 834); const sw = P.sidebar(b, T, 2, { collapsed: true });
  const lw = 400; H.rect(b, 'pane edge', sw + lw, 0, 1, 834, T.border); H.text(b, 'Requests', sw + 24, 44, { size: 30, weight: 700, color: T.fg }); let cx = sw + 24; ['All', 'Pending', 'Approved', 'Available'].forEach((l, i) => { const c = C.chip(b, T, cx, 100, l, { active: i === 0 }); cx += c.width + 8; });
  [[1, 'requested', '2 days ago'], [3, 'downloading', '3 days ago'], [0, 'available', '1 week ago'], [8, 'declined', '2 weeks ago'], [10, 'failed', '3 weeks ago']].forEach(([i, s, w], k) => { const yy = 156 + k * 116; if (k === 1) H.rect(b, 'selected', sw + 8, yy - 8, lw - 16, 112, T.muted, { r: 14 }); C.requestRow(b, T, sw + 20, yy, lw - 40, i, { status: s, when: w }); });
  const dx = sw + lw + 1; H.text(b, 'Severance', dx + 40, 60, { size: 30, weight: 700, color: T.fg }); H.text(b, '2022  ·  TV show', dx + 40, 102, { size: 15, color: T.mutedFg }); C.badge(b, T, dx + 40, 132, 'downloading', {}); C.progress(b, T, dx + 40, 178, 480, 0.62); H.text(b, '62%  ·  about 12 min left', dx + 40, 194, { size: 13, color: T.mutedFg });
  H.text(b, 'Progress', dx + 40, 250, { size: 22, weight: 700, color: T.fg }); C.timeline(b, T, dx + 44, 296, [['Requested', 'Mar 3, 18:02'], ['Approved', 'Mar 3, 18:05'], ['Searching', 'Mar 3, 18:05'], ['Downloading', 'in progress'], ['Available', 'waiting']], 3);
  H.text(b, 'Details', dx + 40, 600, { size: 22, weight: 700, color: T.fg }); [['Requested by', 'You'], ['Quality profile', 'HD - 1080p'], ['Folder', '/tv']].forEach(([k, v], j) => { H.text(b, k, dx + 40, 644 + j * 36, { size: 15, color: T.mutedFg }); H.text(b, v, dx + 40, 644 + j * 36, { size: 15, weight: 500, color: T.fg, w: 480, align: 'right' }); H.rect(b, 'divider', dx + 40, 644 + j * 36 + 28, 480, 1, T.border); });
  C.btn(b, T, dx + 40, 770, 190, 'Cancel request', { v: 'secondary', h: 46 });
  return b;
};
P.library = (T, x, y) => {
  const b = P.board(T, `iPad · Library · ${T.name}`, x, y, 1194, 834); const sw = P.sidebar(b, T, 3); const cx = sw + 32;
  H.text(b, 'Library', cx, 44, { size: 34, weight: 700, color: T.fg, ls: -0.4 }); let ch = cx; [['All', 'layout-grid', 1], ['Movies', 'film', 0], ['TV', 'tv', 0], ['Unwatched', 'eye-off', 0]].forEach(([l, ic, a]) => { const c = C.chip(b, T, ch, 100, l, { active: !!a, icon: ic }); ch += c.width + 8; });
  H.text(b, '485 titles  ·  Recently added', 1194 - 32 - 260, 108, { size: 14, color: T.mutedFg, w: 260, align: 'right' });
  const cols = 6, pw = 124, gap = 17; for (let i = 0; i < 12; i++) { const col = i % cols, row = Math.floor(i / cols); const px = cx + col * (pw + gap), py = 160 + row * 270; const p = C.poster(b, T, px, py, pw, i, { badge: false }); if ([0, 3, 5, 8].includes(i)) C.watchedMark(p, T, pw - 28, 6); H.text(b, C.items[i].t, px, py + 192, { size: 13, weight: 600, color: T.fg, w: pw }); H.text(b, `${C.items[i].y} · ${C.items[i].k === 'tv' ? 'TV' : 'Movie'}`, px, py + 210, { size: 12, color: T.mutedFg, w: pw }); }
  return b;
};
P.portrait = (T, x, y) => {
  const b = P.board(T, `iPad · Discover portrait · ${T.name}`, x, y, 834, 1194); P.statusBar(b, T, 834);
  H.text(b, 'Discover', 32, 56, { size: 34, weight: 700, color: T.fg, ls: -0.4 }); let cx = 32; [['Trending', true, 'trending-up'], ['Upcoming', false, 'calendar'], ['Movies', false, 'film'], ['TV', false, 'tv']].forEach(([l, a, ic]) => { const c = C.chip(b, T, cx, 112, l, { active: a, icon: ic }); cx += c.width + 8; });
  H.text(b, 'Trending this week', 32, 168, { size: 22, weight: 700, color: T.fg }); for (let i = 0; i < 12; i++) { const col = i % 5, row = Math.floor(i / 5); const pw = 144; const px = 32 + col * (pw + 18), py = 212 + row * 276; C.poster(b, T, px, py, pw, i); H.text(b, C.items[i].t, px, py + 220, { size: 13, weight: 600, color: T.fg, w: pw }); H.text(b, `${C.items[i].y}`, px, py + 238, { size: 12, color: T.mutedFg }); }
  const tb = H.board(b, 'tab bar', 117, 1194 - 34 - 70, 600, 70, { fill: T.card, r: 35, stroke: [T.border, 1], clip: false }); [['compass', 'Discover'], ['search', 'Search'], ['list-checks', 'Requests'], ['library', 'Library'], ['user', 'Profile']].forEach(([ic, l], i) => { const on = i === 0; if (on) H.rect(tb, 'active', i * 120 + 8, 8, 104, 54, T.muted, { r: 27 }); H.icon(tb, ic, i * 120 + 48, 12, 24, on ? T.fg : T.mutedFg, on ? 2.2 : 1.8); H.text(tb, l, i * 120, 42, { size: 11, weight: on ? 600 : 500, color: on ? T.fg : T.mutedFg, w: 120, align: 'center' }); });
  return b;
};
P.duo = (T, x, y) => {
  const b = H.board(null, `Foldable inner · Discover + detail · ${T.name}`, x, y, 840, 900, { fill: T.bg, r: 0 }); P.statusBar(b, T, 840);
  const lw = 420; H.rect(b, 'hinge', lw - 1, 0, 2, 900, T.border); H.text(b, 'Discover', 20, 48, { size: 30, weight: 700, color: T.fg }); let cx = 20; [['Trending', true], ['Upcoming', false], ['Movies', false]].forEach(([l, a]) => { const c = C.chip(b, T, cx, 96, l, { active: a }); cx += c.width + 8; });
  for (let i = 0; i < 6; i++) { const col = i % 3, row = Math.floor(i / 3); const pw = 118; const px = 20 + col * (pw + 12), py = 150 + row * 228; C.poster(b, T, px, py, pw, i); H.text(b, C.items[i].t, px, py + 182, { size: 13, weight: 600, color: T.fg, w: pw }); H.text(b, C.items[i].y, px, py + 200, { size: 12, color: T.mutedFg }); }
  const tb = H.board(b, 'tab bar', 20, 900 - 20 - 60, lw - 40, 60, { fill: T.card, r: 30, stroke: [T.border, 1], clip: false }); [['compass', 0], ['search', 1], ['list-checks', 2], ['library', 3], ['user', 4]].forEach(([ic, i]) => { const w = (lw - 40) / 5; if (i === 0) H.rect(tb, 'active', i * w + 6, 6, w - 12, 48, T.muted, { r: 24 }); H.icon(tb, ic, i * w + (w - 24) / 2, 18, 24, i === 0 ? T.fg : T.mutedFg, 2); });
  const dx = lw + 1, dw = 840 - dx, pal = C.pal[0]; H.board(b, 'backdrop', dx, 0, dw, 280, { fill: [H.grad([[pal[0], 0], [pal[1], 0.82], [T.bg, 1]])], clip: true });
  C.poster(b, T, dx + 20, 130, 100, 0, { badge: false, noTitle: true, r: 10 }); H.text(b, 'Dune: Part Two', dx + 136, 150, { size: 24, weight: 700, color: '#FFFFFF', w: dw - 156 }); H.text(b, '2024  ·  2h 46m', dx + 136, 184, { size: 13, color: '#D4D4D8' }); C.badge(b, T, dx + 136, 210, 'available', {});
  C.btn(b, T, dx + 20, 300, dw - 80, 'Open in Jellyfin', { icon: 'play', v: 'success', h: 46 }); const mb = H.board(b, 'menu', dx + dw - 52, 300, 46, 46, { fill: T.muted, r: 23, clip: false }); H.icon(mb, 'ellipsis', 11, 11, 24, T.fg, 2);
  H.text(b, 'Overview', dx + 20, 374, { size: 20, weight: 700, color: T.fg }); H.text(b, 'Paul Atreides unites with Chani and the Fremen while seeking revenge against the conspirators who destroyed his family.', dx + 20, 406, { size: 15, color: T.fg, w: dw - 40, lh: 1.4, op: 0.85 });
  H.text(b, 'Cast', dx + 20, 520, { size: 20, weight: 700, color: T.fg }); [['Timothée Chalamet', 'Paul'], ['Zendaya', 'Chani'], ['Rebecca Ferguson', 'Jessica']].forEach(([n, r], j) => C.avatarCol(b, T, dx + 12 + j * 92, 560, n, r, j + 1));
  return b;
};
P.duoOuter = (T, x, y) => { const b = C.screen(T, `Foldable outer (phone) · ${T.name}`, x, y); b.resize(300, 740); return b; };
S.ipadList = [['iPad · Discover (landscape)', P.discover], ['iPad · Search + detail', P.master], ['iPad · Requests', P.requests], ['iPad · Library', P.library], ['iPad · Discover (portrait)', P.portrait]];
S.duoList = [['Foldable · inner display', P.duo]];
