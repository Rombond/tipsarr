// iPad Profile proposals (landscape with sidebar, portrait with floating tab bar). Needs H, C, S.P (70_ipad.js).
const H = storage.H, C = storage.C, S = storage.S, P = S.P;
P.sec = (b, T, x, y, w, title, action) => { H.text(b, title, x, y, { size: 20, weight: 700, color: T.fg, name: 'section ' + title }); if (action) H.text(b, action, x, y + 3, { size: 15, weight: 500, color: T.mutedFg, w, align: 'right', name: 'action' }); };
P.card = (b, T, name, x, y, w, h, r = 14) => H.board(b, name, x, y, w, h, { fill: T.card, r, stroke: [T.border, 1], clip: false });
P.label = (txt, x, y) => H.text(penpot.root, txt, x, y - 36, { size: 14, weight: 600, color: '#9CA3AF', name: 'label ' + txt });
P.statTile = (b, T, x, y, w, h, icon, value, label, wide) => {
  const c = P.card(b, T, 'stat ' + label, x, y, w, h);
  if (wide) { const ib = H.board(c, 'icon bg', 16, (h - 40) / 2, 40, 40, { fill: T.muted, r: 12, clip: false }); H.icon(ib, icon, 10, 10, 20, T.fg, 2); H.text(c, value, 68, 14, { size: 26, weight: 700, color: T.fg }); H.text(c, label, 68, 50, { size: 13, color: T.mutedFg }); }
  else { H.icon(c, icon, w / 2 - 10, 10, 20, T.mutedFg, 2); H.text(c, value, 0, 34, { size: 24, weight: 700, color: T.fg, w, align: 'center' }); H.text(c, label, 0, 62, { size: 12, color: T.mutedFg, w, align: 'center' }); }
};
P.requestCard = (b, T, x, y, w, i, status, when) => {
  const c = P.card(b, T, 'request ' + C.items[i].t, x, y, w, 84);
  C.poster(c, T, 12, 9, 44, i, { badge: false, noTitle: true, noIcon: true, r: 6 });
  H.text(c, C.items[i].t, 68, 12, { size: 15, weight: 600, color: T.fg, w: w - 80, name: 'title' });
  C.badge(c, T, 68, 40, status, {}); H.text(c, when, 68, 64, { size: 11, color: T.mutedFg });
};
P.adminTile = (b, T, x, y, w, icon, color, label, count) => {
  const c = P.card(b, T, 'admin ' + label, x, y, w, 84);
  const ib = H.board(c, 'icon bg', 14, 14, 34, 34, { fill: color, op: T.isDark ? 0.2 : 0.16, r: 10, clip: false }); H.icon(ib, icon, 8, 8, 18, color, 2.2);
  if (count) H.text(c, count, 0, 18, { size: 20, weight: 700, color: T.fg, w: w - 16, align: 'right' });
  H.text(c, label, 14, 58, { size: 14, weight: 600, color: T.fg });
};
P.carousel = (b, T, x, y, n, pw, gap) => { for (let k = 0; k < n; k++) { const i = [3, 0, 5, 2, 7, 9][k]; const px = x + k * (pw + gap); C.poster(b, T, px, y, pw, i, { badge: false }); H.text(b, C.items[i].t, px, y + Math.round(pw * 1.5) + 8, { size: 13, weight: 600, color: T.fg, w: pw }); H.text(b, ['42 h · 12 plays', '9 h · 3 plays', '31 h · 28 ep.', '18 h · 15 ep.', '7 h · 2 plays', '6 h · 6 ep.'][k], px, y + Math.round(pw * 1.5) + 28, { size: 11, color: T.mutedFg, w: pw }); } };
P.profileLandscape = (T, x, y) => {
  const b = P.board(T, `iPad · Profile (landscape) · ${T.name}`, x, y, 1194, 834); const sw = P.sidebar(b, T, 4);
  const cx = sw + 32, lw = 300, rx = cx + lw + 28, rw = 1194 - rx - 32;
  H.text(b, 'Profile', cx, 44, { size: 34, weight: 700, color: T.fg, ls: -0.4 }); const gb = H.board(b, 'settings button', 1194 - 32 - 44, 46, 44, 44, { fill: T.muted, r: 22, clip: false }); H.icon(gb, 'settings', 11, 11, 22, T.fg, 2);
  const pc = P.card(b, T, 'profile card', cx, 110, lw, 400, 20);
  C.avatarSm(pc, T, lw / 2 - 52, 28, 'Alice', 3, 104); const cam = H.board(pc, 'camera', lw / 2 + 18, 100, 32, 32, { fill: T.primary, r: 16, clip: false, stroke: [T.card, 3] }); H.icon(cam, 'image', 7, 7, 18, T.primaryFg, 2);
  H.text(pc, 'Alice', 0, 148, { size: 28, weight: 700, color: T.fg, w: lw, align: 'center' }); H.text(pc, 'Administrator', 0, 186, { size: 14, color: T.mutedFg, w: lw, align: 'center' });
  H.text(pc, 'Member since Oct 2025  ·  last seen today', 20, 214, { size: 13, color: T.mutedFg, w: lw - 40, align: 'center', lh: 1.35 });
  H.rect(pc, 'divider', 20, 268, lw - 40, 1, T.border); H.icon(pc, 'server', 20, 286, 20, T.mutedFg, 1.8); H.text(pc, 'tipsarr.brebond.com', 52, 284, { size: 14, weight: 600, color: T.fg }); H.text(pc, 'Tipsarr 0.5.0 · connected', 52, 304, { size: 12, color: T.mutedFg });
  C.btn(pc, T, 20, 342, lw - 40, 'Change picture', { v: 'secondary', h: 40, size: 14, icon: 'image' });
  const tw = (lw - 24) / 3; P.statTile(b, T, cx, 530, tw, 84, 'list-checks', '12', 'Requests'); P.statTile(b, T, cx + tw + 12, 530, tw, 84, 'bookmark', '8', 'Watchlist'); P.statTile(b, T, cx + 2 * (tw + 12), 530, tw, 84, 'eye', '143', 'Watched');
  const st = P.card(b, T, 'settings row', cx, 634, lw, 56); H.icon(st, 'settings', 16, 17, 22, T.fg, 2); H.text(st, 'Settings', 52, 16, { size: 17, weight: 500, color: T.fg }); H.icon(st, 'chevron-right', lw - 36, 17, 22, T.mutedFg, 2);
  P.sec(b, T, rx, 110, rw, 'Most watched', 'See all'); P.carousel(b, T, rx, 150, 4, 114, (rw - 4 * 114) / 3);
  P.sec(b, T, rx, 396, rw, 'Recent requests', 'All'); const cw = (rw - 14) / 2;
  P.requestCard(b, T, rx, 436, cw, 1, 'requested', '2 hours ago'); P.requestCard(b, T, rx + cw + 14, 436, cw, 3, 'downloading', 'yesterday'); P.requestCard(b, T, rx, 532, cw, 0, 'available', '3 days ago'); P.requestCard(b, T, rx + cw + 14, 532, cw, 8, 'declined', 'last week');
  H.text(b, 'Administration', rx, 650, { size: 12, weight: 600, color: T.mutedFg, ls: 0.5, upper: true }); const aw = (rw - 3 * 14) / 4;
  [['message-square', '#EA580C', 'Issues', '2'], ['users', '#0EA5E9', 'Users', null], ['refresh-cw', '#D946EF', 'Sync & jobs', null], ['chart-no-axes-column', '#6366F1', 'Stats', null]].forEach(([ic, col, l, n], k) => P.adminTile(b, T, rx + k * (aw + 14), 678, aw, ic, col, l, n));
  return b;
};
P.profilePortrait = (T, x, y) => {
  const b = P.board(T, `iPad · Profile (portrait) · ${T.name}`, x, y, 834, 1194); P.statusBar(b, T, 834);
  const cx = 32, cw = 770;
  H.text(b, 'Profile', cx, 56, { size: 34, weight: 700, color: T.fg, ls: -0.4 }); const gb = H.board(b, 'settings button', 834 - 32 - 44, 58, 44, 44, { fill: T.muted, r: 22, clip: false }); H.icon(gb, 'settings', 11, 11, 22, T.fg, 2);
  const pc = P.card(b, T, 'profile card', cx, 118, cw, 148, 20);
  C.avatarSm(pc, T, 24, 26, 'Alice', 3, 96); const cam = H.board(pc, 'camera', 90, 90, 30, 30, { fill: T.primary, r: 15, clip: false, stroke: [T.card, 3] }); H.icon(cam, 'image', 6, 6, 18, T.primaryFg, 2);
  H.text(pc, 'Alice', 144, 28, { size: 28, weight: 700, color: T.fg }); H.text(pc, 'Administrator  ·  Member since Oct 2025  ·  last seen today', 144, 66, { size: 14, color: T.mutedFg, w: cw - 330 });
  H.icon(pc, 'server', 144, 106, 18, T.mutedFg, 1.8); H.text(pc, 'tipsarr.brebond.com  ·  Tipsarr 0.5.0', 170, 105, { size: 13, color: T.mutedFg });
  C.btn(pc, T, cw - 24 - 170, 54, 170, 'Change picture', { v: 'secondary', h: 40, size: 14, icon: 'image' });
  const tw = (cw - 32) / 3; P.statTile(b, T, cx, 286, tw, 84, 'list-checks', '12', 'Requests', true); P.statTile(b, T, cx + tw + 16, 286, tw, 84, 'bookmark', '8', 'Watchlist', true); P.statTile(b, T, cx + 2 * (tw + 16), 286, tw, 84, 'eye', '143', 'Watched', true);
  P.sec(b, T, cx, 396, cw, 'Most watched', 'See all'); P.carousel(b, T, cx, 436, 6, 116, (cw - 6 * 116) / 5);
  P.sec(b, T, cx, 676, cw, 'Recent requests', 'All'); const rw = (cw - 14) / 2;
  P.requestCard(b, T, cx, 716, rw, 1, 'requested', '2 hours ago'); P.requestCard(b, T, cx + rw + 14, 716, rw, 3, 'downloading', 'yesterday'); P.requestCard(b, T, cx, 812, rw, 0, 'available', '3 days ago'); P.requestCard(b, T, cx + rw + 14, 812, rw, 8, 'declined', 'last week');
  H.text(b, 'Administration', cx, 932, { size: 12, weight: 600, color: T.mutedFg, ls: 0.5, upper: true }); const aw = (cw - 3 * 14) / 4;
  [['message-square', '#EA580C', 'Issues', '2'], ['users', '#0EA5E9', 'Users', null], ['refresh-cw', '#D946EF', 'Sync & jobs', null], ['chart-no-axes-column', '#6366F1', 'Stats', null]].forEach(([ic, col, l, n], k) => P.adminTile(b, T, cx + k * (aw + 14), 960, aw, ic, col, l, n));
  const tb = H.board(b, 'tab bar', 117, 1194 - 34 - 70, 600, 70, { fill: T.card, r: 35, stroke: [T.border, 1], clip: false }); [['compass', 'Discover'], ['search', 'Search'], ['list-checks', 'Requests'], ['library', 'Library'], ['user', 'Profile']].forEach(([ic, l], i) => { const on = i === 4; if (on) H.rect(tb, 'active', i * 120 + 8, 8, 104, 54, T.muted, { r: 27 }); H.icon(tb, ic, i * 120 + 48, 12, 24, on ? T.fg : T.mutedFg, on ? 2.2 : 1.8); H.text(tb, l, i * 120, 42, { size: 11, weight: on ? 600 : 500, color: on ? T.fg : T.mutedFg, w: 120, align: 'center' }); });
  return b;
};
S.ipadProfileList = [['iPad · Profile (landscape)', P.profileLandscape], ['iPad · Profile (portrait)', P.profilePortrait]];
