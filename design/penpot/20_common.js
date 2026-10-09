const H = storage.H, C = storage.C, S = storage.S;
C.screen = (T, name, x, y, o = {}) => { const b = H.board(null, name, x, y, 393, o.h || 852, { fill: T.bg, r: 0 }); if (o.status !== false) C.statusBar(b, T); return b; };
C.sheet = (b, T, h, title, o = {}) => {
  H.rect(b, 'scrim', 0, 0, 393, 852, '#000000', { op: 0.5 });
  const s = H.board(b, 'sheet', 0, 852 - h, 393, h, { fill: T.card, r: 28, clip: false, stroke: [T.border, 1] });
  s.borderRadiusBottomLeft = 0; s.borderRadiusBottomRight = 0;
  H.rect(s, 'grabber', 393 / 2 - 18, 8, 36, 5, T.mutedFg, { r: 3, opacity: 0.5 });
  H.text(s, title, 20, 28, { size: 20, weight: 700, color: T.fg, name: 'sheet title' });
  if (!o.noClose) { const c = H.board(s, 'close', 393 - 20 - 32, 24, 32, 32, { fill: T.muted, r: 16, clip: false }); H.icon(c, 'x', 8, 8, 16, T.fg, 2.2); }
  return s;
};
C.check = (p, T, x, y, on, size = 24) => { if (on) { H.ellipse(p, 'checkbox on', x, y, size, size, T.primary); H.icon(p, 'check', x + 4, y + 4, size - 8, T.primaryFg, 3); } else H.ellipse(p, 'checkbox off', x, y, size, size, null, { stroke: [T.mutedFg, 1.5, 0.7] }); };
C.progress = (p, T, x, y, w, pct, color = '#6366F1') => { H.rect(p, 'progress track', x, y, w, 6, T.muted, { r: 3 }); H.rect(p, 'progress', x, y, Math.max(6, w * pct), 6, color, { r: 3 }); };
C.pill = (p, T, x, y, label, o = {}) => { const w = label.length * 6.6 + 20; const b = H.board(p, 'pill ' + label, x, y, w, 24, { fill: T.muted, r: 12, clip: false }); H.text(b, label, 10, 4, { size: 12, weight: 600, color: o.color || T.fg }); return b; };
C.avatarCol = (p, T, x, y, name, role, ci) => { const pal = C.pal[ci % C.pal.length]; H.ellipse(p, 'cast avatar', x + 4, y, 64, 64, [H.grad([[pal[0], 0], [pal[1], 1]])]); H.text(p, name.split(' ').map(s => s[0]).join(''), x + 4, y + 20, { size: 18, weight: 600, color: '#FFFFFF', w: 64, align: 'center' }); H.text(p, name, x, y + 72, { size: 12, weight: 600, color: T.fg, w: 72, align: 'center', lh: 1.15 }); H.text(p, role, x, y + 100, { size: 11, color: T.mutedFg, w: 72, align: 'center' }); };
C.optionRow = (p, T, x, y, w, icon, label, value, o = {}) => {
  const r = H.board(p, 'option ' + label, x, y, w, 56, { fill: T.muted, r: 14, clip: false });
  H.icon(r, icon, 14, 17, 22, T.mutedFg, 1.8); H.text(r, label, 46, 9, { size: 12, weight: 600, color: T.mutedFg }); H.text(r, value, 46, 26, { size: 16, weight: 500, color: T.fg });
  H.icon(r, o.chev || 'chevrons-up-down', w - 34, 17, 20, T.mutedFg, 2); return r;
};
C.row = (p, T, x, y, w, o = {}) => {
  // generic grouped-list row: icon, title, optional subtitle, optional value, chevron
  const h = o.sub ? 64 : 52; const r = H.board(p, 'row ' + o.title, x, y, w, h, { fill: o.card === false ? null : T.card, r: 0, clip: false });
  if (o.icon) { const ib = H.board(r, 'icon bg', 14, (h - 32) / 2, 32, 32, { fill: o.iconBg || T.muted, op: o.iconBgOp ?? 1, r: 9, clip: false }); H.icon(ib, o.icon, 6, 6, 20, o.iconColor || T.fg, 1.9); }
  const tx = o.icon ? 60 : 16;
  H.text(r, o.title, tx, o.sub ? 12 : 15, { size: 16, weight: 500, color: o.danger ? T.destructive : T.fg, w: w - tx - (o.value ? 120 : 44) });
  if (o.sub) H.text(r, o.sub, tx, 34, { size: 13, color: T.mutedFg, w: w - tx - 44 });
  if (o.value) H.text(r, o.value, w - 150, o.sub ? 22 : 16, { size: 15, color: T.mutedFg, w: 112, align: 'right' });
  if (o.toggle !== undefined) { const on = o.toggle; const tg = H.board(r, 'toggle', w - 16 - 51, (h - 31) / 2, 51, 31, { fill: on ? '#10B981' : (T.isDark ? '#3A3A3A' : '#D4D4D4'), r: 16, clip: false }); H.ellipse(tg, 'knob', on ? 22 : 2, 2, 27, 27, '#FFFFFF'); }
  else if (o.chev !== false) H.icon(r, 'chevron-right', w - 34, (h - 20) / 2, 20, T.mutedFg, 2);
  return r;
};
C.group = (p, T, x, y, w, rows, o = {}) => {
  // rounded card containing rows, with dividers; returns board
  const hs = rows.map(r => r.sub ? 64 : 52); const total = hs.reduce((a, b) => a + b, 0);
  const g = H.board(p, o.name || 'group', x, y, w, total, { fill: T.card, r: 16, clip: true, stroke: [T.border, 1] });
  let yy = 0; rows.forEach((r, i) => { C.row(g, T, 0, yy, w, { ...r, card: false }); yy += hs[i]; if (i < rows.length - 1) H.rect(g, 'divider', r.icon ? 60 : 16, yy - 0.5, w - (r.icon ? 60 : 16), 1, T.border); });
  return g;
};
S.heroNav = (b, T, o = {}) => {
  const mk = (ic, x) => { const c = H.board(b, 'nav ' + ic, x, 56, 40, 40, { fill: '#000000', op: 0.35, r: 20, clip: false }); H.icon(c, ic, 9, 9, 22, '#FFFFFF', 2); };
  mk('chevron-left', 16); mk('bookmark', 393 - 16 - 40 - 48); mk('share', 393 - 16 - 40);
};
S.navBar = (b, T, title, o = {}) => { H.icon(b, 'chevron-left', 14, 64, 26, T.fg, 2.2); H.text(b, title, 0, 66, { size: 17, weight: 600, color: T.fg, w: 393, align: 'center', name: 'nav title' }); if (o.right) H.text(b, o.right, 300, 67, { size: 17, weight: 500, color: T.fg, w: 77, align: 'right', name: 'nav action' }); };
// layout a slice of a list for both themes; positions by global index so slices tile together; stops after budgetMs and returns the next index
S.layoutPart = (list, o = {}) => {
  const t0 = Date.now(); const budget = o.budgetMs || 70000;
  const cols = o.cols || 6, yBase = o.yBase || 0, rowH = o.rowH || 1050, tag = o.tag || '', from = o.from || 0, to = o.to ?? list.length, total = o.total ?? list.length;
  const rows = Math.ceil(total / cols); const themes = o.themes || [C.themes.light, C.themes.dark];
  storage.ids = storage.ids || {}; let i = from;
  for (; i < to; i++) {
    if (Date.now() - t0 > budget) break;
    const [label, fn] = list[i]; const col = i % cols, rowi = Math.floor(i / cols);
    themes.forEach((T, ri) => { const row = rowi + ri * rows; const x = col * 460, y = yBase + row * rowH; const b = fn(T, x, y); S.label(`${label} · ${T.name}`, x, y); storage.ids[`${tag}${i}_${T.name}`] = b.id; });
  }
  return { next: i, done: i >= to, ms: Date.now() - t0 };
};
// snapshot every top-level board of the current page whose name matches re (default all) to the local receiver, as <prefix><n>
S.snapPage = async (prefix, re = null, wait = 1500) => { await new Promise(r => setTimeout(r, wait)); const out = []; for (const b of penpot.root.children) { if (b.type !== 'board') continue; if (re && !re.test(b.name)) continue; const m = penpot.generateMarkup([b], { type: 'svg' }); const key = prefix + b.name.replace(/[^A-Za-z0-9]+/g, '_'); await fetch('http://127.0.0.1:8765/' + key, { method: 'POST', body: m }); out.push(key); } return out; };
S.clearPage = () => { for (const s of [...penpot.root.children]) s.remove(); storage.ids = {}; };
S.go = async (name) => { await penpot.openPage(storage.pageIds[name]); };
storage.pageIds = {}; penpotUtils.getPages().forEach(p => { storage.pageIds[p.name] = p.id; });
// wide layout for iPad boards: items in a column per theme row
S.layoutWide = (list, o = {}) => { const t0 = Date.now(); const budget = o.budgetMs || 65000; const cols = o.cols || 2, colW = o.colW || 1300, rowH = o.rowH || 1350, yBase = o.yBase || 0, from = o.from || 0, to = o.to ?? list.length, tag = o.tag || 'w'; const rows = Math.ceil(list.length / cols); storage.ids = storage.ids || {}; let i = from;
  for (; i < to; i++) { if (Date.now() - t0 > budget) break; const [label, fn] = list[i]; const col = i % cols, rowi = Math.floor(i / cols); [C.themes.light, C.themes.dark].forEach((T, ri) => { const x = col * colW, y = yBase + (rowi + ri * rows) * rowH; const b = fn(T, x, y); S.label(`${label} · ${T.name}`, x, y); storage.ids[`${tag}${i}_${T.name}`] = b.id; }); }
  return { next: i, done: i >= to, ms: Date.now() - t0 }; };
