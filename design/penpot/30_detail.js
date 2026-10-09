const H = storage.H, C = storage.C, S = storage.S;
S.detailAction = (b, T, y, st, kind) => {
  if (st === 'none') C.btn(b, T, 16, y, 295, kind === 'tv' ? 'Request…' : 'Request', { icon: 'plus', h: 50 });
  if (st === 'requested') C.btn(b, T, 16, y, 295, 'Requested', { icon: 'clock', h: 50, v: 'secondary' });
  if (st === 'approved') C.btn(b, T, 16, y, 295, 'Approved', { icon: 'thumbs-up', h: 50, v: 'secondary' });
  if (st === 'downloading') { C.btn(b, T, 16, y, 295, 'Downloading', { icon: 'download', h: 50, v: 'secondary' }); C.progress(b, T, 16, y + 66, 361, 0.62); H.text(b, '62%  ·  about 12 min left', 16, y + 80, { size: 13, color: T.mutedFg }); }
  if (st === 'available') C.btn(b, T, 16, y, 295, 'Open in Jellyfin', { icon: 'play', h: 50, v: 'success' });
  if (st === 'declined') C.btn(b, T, 16, y, 295, 'Request again', { icon: 'rotate-ccw', h: 50, v: 'secondary' });
  if (st === 'failed') C.btn(b, T, 16, y, 295, 'Retry', { icon: 'refresh-cw', h: 50, v: 'primary' });
  const b2 = H.board(b, 'menu', 321, y, 50, 50, { fill: T.muted, r: 25, clip: false }); H.icon(b2, 'ellipsis', 13, 13, 24, T.fg, 2);
};
const synopsis = 'Paul Atreides unites with Chani and the Fremen while seeking revenge against the conspirators who destroyed his family. Facing a choice between the love of his life and the fate of the known universe, he endeavors to prevent a terrible future only he can foresee.';
S.detailFull = (T, x, y, o = {}) => {
  const kind = o.kind || 'movie', st = o.st || 'none', tall = o.tall; const i = o.i ?? (kind === 'tv' ? 3 : 0); const it = C.items[i];
  const b = C.screen(T, `${kind === 'tv' ? 'TV' : 'Movie'} detail · ${st}${tall ? ' · full scroll' : ''} · ${T.name}`, x, y, { h: tall ? (kind === 'tv' ? 1880 : 1560) : 852, status: false });
  const pal = C.pal[i];
  H.board(b, 'backdrop', 0, 0, 393, 300, { fill: [H.grad([[pal[0], 0], [pal[1], 0.65], [T.bg, 1]])], clip: true });
  C.statusBar(b, { ...T, fg: '#FFFFFF' }); S.heroNav(b, T);
  C.poster(b, T, 16, 186, 120, i, { badge: false, noTitle: true, r: 12 });
  H.text(b, it.t, 152, 222, { size: 24, weight: 700, color: T.fg, w: 225, lh: 1.15, name: 'title', ls: -0.2 });
  H.text(b, kind === 'tv' ? `${it.y}  ·  3 seasons  ·  TV-MA` : `${it.y}  ·  2h 46m  ·  PG-13`, 152, 290, { size: 13, color: T.mutedFg, w: 225, name: 'meta' });
  const rb = H.board(b, 'rating', 152, 316, 54, 26, { fill: T.muted, r: 13, clip: false }); H.icon(rb, 'star', 8, 6, 14, '#F59E0B', 2, true); H.text(rb, it.r.toFixed(1), 27, 5, { size: 13, weight: 600, color: T.fg });
  const stKey = { none: it.s, requested: 'requested', approved: 'approved', downloading: 'downloading', available: 'available', declined: 'declined', failed: 'failed' }[st];
  if (st !== 'none' && stKey) C.badge(b, T, 214, 318, stKey, {});
  S.detailAction(b, T, 396, st, kind);
  const off = st === 'downloading' ? 46 : 0;
  if (st === 'declined') C.banner(b, T, 16, 462, 361, 'error', 'Declined by an admin: not available in this quality.');
  if (st === 'failed') C.banner(b, T, 16, 462, 361, 'warn', 'The download failed. An admin can retry it.');
  const base = 470 + off + (st === 'declined' || st === 'failed' ? 62 : 0);
  H.text(b, 'Overview', 16, base, { size: 20, weight: 700, color: T.fg });
  H.text(b, synopsis, 16, base + 34, { size: 15, color: T.fg, w: 361, lh: 1.4, op: 0.85, name: 'overview' });
  let cx = 16; ['Sci-Fi', 'Adventure', 'Drama'].forEach(g => { const c = C.chip(b, T, cx, base + 164, g, {}); cx += c.width + 8; });
  if (tall) {
    let y0 = base + 224;
    if (kind === 'tv') {
      H.text(b, 'Seasons', 16, y0, { size: 20, weight: 700, color: T.fg }); y0 += 40;
      [['Season 1', '9 episodes · 2022', 'available'], ['Season 2', '10 episodes · 2025', 'downloading'], ['Season 3', '10 episodes · 2026', null]].forEach(([s, m, k], j) => { const r = H.board(b, 'season ' + (j + 1), 16, y0 + j * 72, 361, 64, { fill: T.card, r: 14, stroke: [T.border, 1], clip: false }); H.text(r, s, 16, 12, { size: 16, weight: 600, color: T.fg }); H.text(r, m, 16, 34, { size: 13, color: T.mutedFg }); if (k) { const bd = C.badge(r, T, 0, 20, k, {}); bd.x = r.x + 361 - 44 - bd.width; } H.icon(r, 'chevron-right', 361 - 34, 20, 22, T.mutedFg, 2); });
      y0 += 3 * 72 + 24;
    }
    H.text(b, 'Ratings', 16, y0, { size: 20, weight: 700, color: T.fg }); y0 += 36;
    [['TMDB', '8.2'], ['IMDb', '8.5'], ['Rotten Tomatoes', '92%'], ['Metacritic', '79']].forEach(([s, v], j) => { const c = H.board(b, 'rating ' + s, 16 + (j % 2) * 184, y0 + Math.floor(j / 2) * 56, 177, 48, { fill: T.muted, r: 12, clip: false }); H.text(c, s, 12, 8, { size: 11, weight: 600, color: T.mutedFg, upper: true, ls: 0.3 }); H.text(c, v, 12, 24, { size: 16, weight: 700, color: T.fg }); });
    y0 += 124;
    H.text(b, 'Details', 16, y0, { size: 20, weight: 700, color: T.fg }); y0 += 36;
    [['Status', 'Released'], ['Release date', 'March 1, 2024'], ['Runtime', '2h 46m'], ['Original language', 'English'], [kind === 'tv' ? 'Network' : 'Budget', kind === 'tv' ? 'Apple TV+' : '$190,000,000']].forEach(([k, v], j) => { H.text(b, k, 16, y0 + j * 34, { size: 15, color: T.mutedFg }); H.text(b, v, 16, y0 + j * 34, { size: 15, weight: 500, color: T.fg, w: 361, align: 'right' }); H.rect(b, 'divider', 16, y0 + j * 34 + 26, 361, 1, T.border); });
    y0 += 5 * 34 + 24;
    H.text(b, 'Cast', 16, y0, { size: 20, weight: 700, color: T.fg }); y0 += 40;
    [['Timothée Chalamet', 'Paul'], ['Zendaya', 'Chani'], ['Rebecca Ferguson', 'Jessica'], ['Josh Brolin', 'Gurney'], ['Austin Butler', 'Feyd']].forEach(([n, r], j) => C.avatarCol(b, T, 12 + j * 78, y0, n, r, j + 1));
    y0 += 150;
    H.text(b, 'More like this', 16, y0, { size: 20, weight: 700, color: T.fg }); y0 += 38;
    [4, 5, 6, 7].forEach((n, j) => C.poster(b, T, 16 + j * 126, y0, 110, n));
    y0 += 190;
    const iss = H.board(b, 'report issue row', 16, y0, 361, 52, { fill: T.card, r: 14, stroke: [T.border, 1], clip: false }); H.icon(iss, 'flag', 16, 14, 22, T.mutedFg, 1.8); H.text(iss, 'Report an issue', 52, 15, { size: 16, weight: 500, color: T.fg }); H.icon(iss, 'chevron-right', 361 - 34, 15, 22, T.mutedFg, 2);
  } else C.homeIndicator(b, T);
  return b;
};
const base = (T, x, y, name, st = 'none', kind = 'movie', i = 0) => { const b = S.detailFull(T, x, y, { kind, st, i }); b.name = name; return b; };
S.requestSheet = (T, x, y, o = {}) => {
  const b = base(T, x, y, `Request sheet · ${o.name} · ${T.name}`, 'none', 'movie', 0);
  const h = o.folder ? 500 : 430; const s = C.sheet(b, T, h, 'Request');
  C.poster(s, T, 20, 76, 56, 0, { badge: false, noTitle: true, noIcon: true, r: 8 });
  H.text(s, 'Dune: Part Two', 92, 80, { size: 17, weight: 600, color: T.fg }); H.text(s, '2024  ·  Movie', 92, 104, { size: 14, color: T.mutedFg });
  C.optionRow(s, T, 20, 176, 353, 'sliders-horizontal', 'Quality profile', 'HD - 1080p');
  if (o.folder) C.optionRow(s, T, 20, 244, 353, 'hard-drive', 'Folder', '/movies  ·  1.2 TB free');
  H.text(s, o.note, 24, o.folder ? 312 : 244, { size: 13, color: T.mutedFg, w: 345, lh: 1.35 });
  C.btn(s, T, 20, h - 34 - 52 - 52 - 6, 353, 'Request', { h: 52, icon: 'plus' });
  C.btn(s, T, 20, h - 34 - 52, 353, 'Cancel', { h: 46, v: 'ghost' });
  return b;
};
S.requestTV = (T, x, y) => {
  const b = base(T, x, y, `Request sheet · TV seasons · ${T.name}`, 'none', 'tv', 3);
  const h = 700; const s = C.sheet(b, T, h, 'Request seasons');
  H.text(s, 'Severance', 20, 62, { size: 15, color: T.mutedFg });
  const row = (yy, label, meta, on, dis, badge) => { const r = H.board(s, 'season row ' + label, 20, yy, 353, 60, { fill: T.muted, r: 14, clip: false }); if (dis) r.opacity = 0.6; C.check(r, T, 14, 18, on); H.text(r, label, 52, 11, { size: 16, weight: 600, color: T.fg }); H.text(r, meta, 52, 33, { size: 13, color: T.mutedFg }); if (badge) { const bd = C.badge(r, T, 0, 19, badge, {}); bd.x = r.x + 353 - 14 - bd.width; } return r; };
  row(96, 'All seasons', '3 seasons', false, false); row(164, 'Season 1', '9 episodes · available', false, true, 'available'); row(232, 'Season 2', '10 episodes · 2025', true, false); row(300, 'Season 3', '10 episodes · 2026', true, false);
  C.optionRow(s, T, 20, 380, 353, 'sliders-horizontal', 'Quality profile', 'HD - 1080p'); C.optionRow(s, T, 20, 448, 353, 'hard-drive', 'Folder', '/tv  ·  3.4 TB free');
  C.btn(s, T, 20, h - 34 - 52 - 52 - 6, 353, 'Request 2 seasons', { h: 52, icon: 'plus' }); C.btn(s, T, 20, h - 34 - 52, 353, 'Cancel', { h: 46, v: 'ghost' });
  return b;
};
S.pickerSheet = (T, x, y) => {
  const b = base(T, x, y, `Quality profile picker · ${T.name}`, 'none', 'movie', 0);
  const s = C.sheet(b, T, 420, 'Quality profile');
  [['Any', false], ['SD - 480p', false], ['HD - 720p', false], ['HD - 1080p', true], ['Ultra-HD - 4K', false]].forEach(([l, on], i) => { const yy = 80 + i * 56; H.text(s, l, 24, yy + 16, { size: 17, weight: on ? 600 : 400, color: T.fg }); if (on) H.icon(s, 'check', 393 - 24 - 22, yy + 15, 22, T.fg, 2.4); H.rect(s, 'divider', 24, yy + 55, 345, 1, T.border); });
  return b;
};
S.cancelConfirm = (T, x, y) => {
  const b = base(T, x, y, `Cancel request confirm · ${T.name}`, 'requested', 'movie', 1);
  H.rect(b, 'scrim', 0, 0, 393, 852, '#000000', { op: 0.5 });
  const g1 = H.board(b, 'action group', 8, 852 - 34 - 170, 377, 106, { fill: T.card, r: 22, stroke: [T.border, 1], clip: false });
  H.text(g1, 'Cancel your request for Oppenheimer?', 16, 16, { size: 13, color: T.mutedFg, w: 345, align: 'center' });
  H.rect(g1, 'divider', 0, 44, 377, 1, T.border); H.text(g1, 'Cancel request', 0, 62, { size: 18, weight: 500, color: T.destructive, w: 377, align: 'center' });
  const g2 = H.board(b, 'keep', 8, 852 - 34 - 56, 377, 56, { fill: T.card, r: 22, stroke: [T.border, 1], clip: false }); H.text(g2, 'Keep request', 0, 16, { size: 18, weight: 600, color: T.fg, w: 377, align: 'center' });
  return b;
};
S.reportSheet = (T, x, y) => {
  const b = base(T, x, y, `Report issue · ${T.name}`, 'available', 'movie', 0);
  const s = C.sheet(b, T, 560, 'Report an issue');
  H.text(s, 'What’s wrong with Dune: Part Two?', 20, 62, { size: 15, color: T.mutedFg });
  let cx = 20; [['Video', true], ['Audio', false], ['Subtitles', false], ['Other', false]].forEach(([l, a]) => { const c = C.chip(s, T, cx, 100, l, { active: a }); cx += c.width + 8; });
  const ta = H.board(s, 'textarea', 20, 156, 353, 150, { fill: T.muted, r: 14, clip: false }); H.text(ta, 'Describe the problem (what, where, when)…', 16, 14, { size: 16, color: T.mutedFg, w: 321 });
  H.text(s, 'An admin will be notified and can reply here.', 24, 316, { size: 13, color: T.mutedFg });
  C.btn(s, T, 20, 560 - 34 - 52 - 52 - 6, 353, 'Send report', { h: 52, icon: 'flag' }); C.btn(s, T, 20, 560 - 34 - 52, 353, 'Cancel', { h: 46, v: 'ghost' });
  return b;
};
S.seasonDetail = (T, x, y) => {
  const b = C.screen(T, `Season detail · ${T.name}`, x, y);
  S.navBar(b, T, 'Season 2');
  H.text(b, 'Severance', 16, 112, { size: 15, color: T.mutedFg }); H.text(b, '10 episodes  ·  2025', 16, 134, { size: 13, color: T.mutedFg });
  C.badge(b, T, 16, 164, 'downloading', {}); C.progress(b, T, 16, 200, 361, 0.4);
  [['Hello, Ms. Cobel', '50m', 'Jan 17', true], ['Goodbye, Mrs. Selvig', '48m', 'Jan 24', true], ['Who Is Alive?', '52m', 'Jan 31', false], ['Woe’s Hollow', '49m', 'Feb 7', false], ['Trojan’s Horse', '51m', 'Feb 14', false]].forEach(([t, m, d, ok], i) => { const yy = 226 + i * 112; const pal = C.pal[(i + 3) % 12]; const th = H.board(b, 'still', 16, yy, 120, 68, { fill: [H.grad([[pal[0], 0], [pal[1], 1]])], r: 10, clip: true }); H.icon(th, 'play', 46, 22, 24, '#FFFFFF', 1.8, true).opacity = 0.7; H.text(b, `E${i + 1}  ${t}`, 148, yy + 2, { size: 15, weight: 600, color: T.fg, w: 200, name: 'ep title' }); H.text(b, `${d}  ·  ${m}`, 148, yy + 24, { size: 12, color: T.mutedFg }); H.text(b, 'A short episode synopsis wraps here over two lines before being cut off with an ellipsis…', 148, yy + 42, { size: 12, color: T.mutedFg, w: 229, lh: 1.3 }); if (ok) H.icon(b, 'circle-check', 353, yy + 2, 22, '#10B981', 2); else H.icon(b, 'clock', 353, yy + 2, 22, T.mutedFg, 2); H.rect(b, 'divider', 16, yy + 100, 361, 1, T.border); });
  C.homeIndicator(b, T); return b;
};
S.personScreen = (T, x, y) => {
  const b = C.screen(T, `Person · ${T.name}`, x, y);
  H.icon(b, 'chevron-left', 14, 64, 26, T.fg, 2.2);
  const pal = C.pal[2]; H.ellipse(b, 'avatar', 393 / 2 - 52, 108, 104, 104, [H.grad([[pal[0], 0], [pal[1], 1]])]); H.text(b, 'Z', 393 / 2 - 52, 140, { size: 40, weight: 700, color: '#FFFFFF', w: 104, align: 'center' });
  H.text(b, 'Zendaya', 0, 230, { size: 28, weight: 700, color: T.fg, w: 393, align: 'center' }); H.text(b, 'Acting  ·  Born 1996', 0, 268, { size: 14, color: T.mutedFg, w: 393, align: 'center' });
  H.text(b, 'Zendaya Maree Stoermer Coleman is an American actress and singer. She started her career as a child model and backup dancer…', 16, 308, { size: 15, color: T.fg, w: 361, lh: 1.4, op: 0.85 }); H.text(b, 'More', 16, 378, { size: 15, weight: 600, color: T.fg });
  H.text(b, 'Known for', 16, 424, { size: 20, weight: 700, color: T.fg });
  for (let i = 0; i < 6; i++) { const col = i % 3, row = Math.floor(i / 3); const px = 16 + col * 121, py = 462 + row * 214; C.poster(b, T, px, py, 112, i); H.text(b, C.items[i].t, px, py + 172, { size: 13, weight: 600, color: T.fg, w: 112 }); }
  C.homeIndicator(b, T); return b;
};
S.collectionScreen = (T, x, y) => {
  const b = C.screen(T, `Collection · ${T.name}`, x, y, { status: false });
  const pal = C.pal[0]; H.board(b, 'backdrop', 0, 0, 393, 260, { fill: [H.grad([[pal[0], 0], [pal[1], 0.6], [T.bg, 1]])], clip: true }); C.statusBar(b, { ...T, fg: '#FFFFFF' }); S.heroNav(b, T);
  H.text(b, 'Dune Collection', 16, 200, { size: 28, weight: 700, color: T.fg, ls: -0.3 }); H.text(b, '2 movies  ·  1 available', 16, 238, { size: 14, color: T.mutedFg });
  H.text(b, 'A saga of power, prophecy and the spice melange on the desert planet Arrakis.', 16, 270, { size: 15, color: T.fg, w: 361, lh: 1.4, op: 0.85 });
  [0, 8].forEach((n, i) => { C.resultRow(b, T, 16, 350 + i * 128, 361, n, { overview: 'A mythic and emotionally charged hero’s journey.' }); H.rect(b, 'divider', 16, 350 + i * 128 + 114, 361, 1, T.border); });
  C.btn(b, T, 16, 640, 361, 'Request missing', { icon: 'plus', h: 50 });
  C.homeIndicator(b, T); return b;
};
S.detailLoading = (T, x, y) => {
  const b = C.screen(T, `Detail · loading · ${T.name}`, x, y, { status: false });
  H.rect(b, 'backdrop skeleton', 0, 0, 393, 300, T.muted, {}); C.statusBar(b, T); S.heroNav(b, T);
  S.skel(b, T, 16, 186, 120, 180, 12); S.skel(b, T, 152, 226, 200, 24, 6); S.skel(b, T, 152, 262, 140, 24, 6); S.skel(b, T, 152, 296, 100, 14, 5);
  S.skel(b, T, 16, 396, 295, 50, 25); S.skel(b, T, 321, 396, 50, 50, 25);
  S.skel(b, T, 16, 480, 120, 20, 6); for (let i = 0; i < 5; i++) S.skel(b, T, 16, 516 + i * 24, i === 4 ? 200 : 361, 14, 5);
  C.homeIndicator(b, T); return b;
};
S.detailTalls = [['Movie · full scroll', (T, x, y) => S.detailFull(T, x, y, { kind: 'movie', st: 'none', tall: true })], ['TV · full scroll', (T, x, y) => S.detailFull(T, x, y, { kind: 'tv', st: 'none', tall: true, i: 3 })]];
S.detailRest = [];
for (const [st, nm, i, kind] of [['requested', 'Requested', 1, 'movie'], ['approved', 'Approved', 2, 'tv'], ['downloading', 'Downloading', 3, 'tv'], ['available', 'Available', 0, 'movie'], ['declined', 'Declined', 8, 'movie'], ['failed', 'Failed', 10, 'movie']]) S.detailRest.push([`State · ${nm}`, (T, x, y) => S.detailFull(T, x, y, { kind, st, i })]);
S.detailRest.push(['Request sheet · admin', (T, x, y) => S.requestSheet(T, x, y, { name: 'with folder', folder: true, note: 'Requests from admins are approved automatically.' })]);
S.detailRest.push(['Request sheet · user', (T, x, y) => S.requestSheet(T, x, y, { name: 'no folder', folder: false, note: 'An admin will approve your request. You’ll be notified when it’s available.' })]);
S.detailRest.push(['Request sheet · TV seasons', S.requestTV], ['Quality profile picker', S.pickerSheet], ['Cancel request confirm', S.cancelConfirm], ['Report an issue', S.reportSheet], ['Season detail', S.seasonDetail], ['Person', S.personScreen], ['Collection', S.collectionScreen], ['Detail · loading', S.detailLoading]);
