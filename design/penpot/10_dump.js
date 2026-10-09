const H = storage.H = storage.H || {}; const C = storage.C = storage.C || {}; const S = storage.S = storage.S || {};
C.appIcon = (p, x, y, size, mode = 'Dark') => {
  const b = H.board(p, 'app icon', x, y, size, size, { r: size * 0.2237, clip: true });
  const g = penpot.createShapeFromSvg(C.iconSvg(mode, size)); b.appendChild(g); g.x = b.x; g.y = b.y; return b;
};
C.badge = (p, T, x, y, key, o = {}) => {
  const s = C.status[key]; const h = o.h || 22; const solid = o.solid;
  const lw = (o.iconOnly ? 0 : s.label.length * 6.3 + 6);
  const w = o.iconOnly ? h : 8 + 12 + lw + 6;
  const b = H.board(p, 'badge ' + key, x, y, w, h, { fill: solid ? s.c : s.c, op: solid ? 1 : 0.16, r: h / 2, clip: false });
  if (!solid) { b.fills = H.fill(s.c, T.isDark ? 0.2 : 0.15); }
  const fg = solid ? '#FFFFFF' : (T.isDark ? s.c : s.c);
  H.icon(b, s.icon, o.iconOnly ? (h - 12) / 2 : 7, (h - 12) / 2, 12, fg, 2.4);
  if (!o.iconOnly) H.text(b, s.label, 22, (h - 14) / 2 + 0.5, { size: 11, weight: 600, color: fg });
  return b;
};
C.banner = (p, T, x, y, w, kind, text) => {
  const col = kind === 'error' ? T.destructive : kind === 'warn' ? '#F59E0B' : '#0EA5E9';
  const ic = kind === 'error' ? 'circle-alert' : kind === 'warn' ? 'triangle-alert' : 'info';
  const lines = Math.ceil(text.length / 44); const h = 24 + lines * 19;
  const b = H.board(p, 'banner ' + kind, x, y, w, h, { fill: col, op: 0.14, r: 14, clip: false });
  H.icon(b, ic, 14, 14, 20, col, 2); H.text(b, text, 44, 14, { size: 14, color: T.fg, w: w - 58, lh: 1.3, name: 'banner text' });
  return b;
};
C.btn = (p, T, x, y, w, label, o = {}) => {
  const h = o.h || 50; const v = o.v || 'primary';
  const fills = { primary: [T.primary, null], secondary: [T.muted, null], ghost: [null, null], outline: [null, [T.border, 1]], destructive: [T.destructive, null], success: ['#10B981', null] };
  const fg = { primary: T.primaryFg, secondary: T.fg, ghost: T.fg, outline: T.fg, destructive: '#FFFFFF', success: '#FFFFFF' }[v];
  const [f, st] = fills[v];
  const b = H.board(p, 'button ' + label, x, y, w, h, { fill: f, r: o.r ?? h / 2, stroke: st, clip: false });
  const iconW = o.icon ? 22 : 0; const tw = label.length * 8.6;
  const sx = (w - tw - iconW) / 2;
  if (o.icon) H.icon(b, o.icon, sx, (h - 18) / 2, 18, fg, 2.2);
  H.text(b, label, sx + iconW, (h - 20) / 2, { size: o.size || 16, weight: 600, color: fg, name: 'label' });
  if (o.disabled) b.opacity = 0.45;
  return b;
};
C.chip = (p, T, x, y, label, o = {}) => {
  const w = label.length * 7.4 + 28 + (o.icon ? 20 : 0);
  const b = H.board(p, 'chip ' + label, x, y, w, 34, { fill: o.active ? T.primary : T.muted, r: 17, clip: false });
  if (o.icon) H.icon(b, o.icon, 12, 9, 16, o.active ? T.primaryFg : T.mutedFg, 2);
  H.text(b, label, (o.icon ? 32 : 14), 7.5, { size: 14, weight: 500, color: o.active ? T.primaryFg : T.fg });
  return b;
};
C.field = (p, T, x, y, w, label, o = {}) => {
  H.text(p, label, x + 4, y, { size: 13, weight: 600, color: o.error ? T.destructive : T.mutedFg, name: 'field label' });
  const f = H.board(p, 'field ' + label, x, y + 22, w, 52, { fill: T.muted, r: 14, clip: false, stroke: o.error ? [T.destructive, 1.5] : (o.focus ? [T.fg, 1.5] : null) });
  if (o.icon) H.icon(f, o.icon, 14, 15, 22, T.mutedFg, 1.8);
  const tx = o.icon ? 46 : 16;
  if (o.secure && o.value) { for (let i = 0; i < o.value.length; i++) H.ellipse(f, 'dot', tx + i * 12, 22, 8, 8, T.fg); }
  else H.text(f, o.value || o.placeholder || '', tx, 15, { size: 17, color: o.value ? T.fg : T.mutedFg, name: 'value' });
  if (o.focus) H.rect(f, 'caret', tx + (o.value ? o.value.length * 9.5 : 0), 14, 2, 24, T.fg, { r: 1 });
  if (o.secure) H.icon(f, 'eye', w - 40, 15, 22, T.mutedFg, 1.8);
  if (o.error && o.msg) H.text(p, o.msg, x + 4, y + 82, { size: 13, color: T.destructive, w: w - 8, name: 'field error' });
  return f;
};
C.finish = (b, T, o = {}) => { if (o.tab != null) C.tabBar(b, T, o.tab); if (o.home !== false) C.homeIndicator(b, T); return b; };
C.fx = (y, op = 1) => y;
C.homeIndicator = (p, T) => H.rect(p, 'home indicator', 129.5, 852 - 13, 134, 5, T.fg, { r: 3 });
C.iconSvg = (mode, size) => {
  const k = size / 1024; const f = (n) => +(n * k).toFixed(3);
  const col = { Dark: { back: '#3A3A42', mid: '#5A5A66', front: '#FAFAFA', pl: '#0A0A0A', g: ['#34343B', '#08080A'] }, Light: { back: '#9C9CA8', mid: '#70707E', front: '#0A0A0A', pl: '#FAFAFA', g: ['#E4E4EA', '#B4B4C0'] }, Tinted: { back: '#4A4A4A', mid: '#8A8A8A', front: '#FFFFFF', pl: '#000000', g: ['#242424', '#000000'] } }[mode];
  const rr = (rot, cx, cy, x, y, w, h, rx, fill) => `<g transform="rotate(${rot} ${f(cx)} ${f(cy)})"><rect x="${f(x)}" y="${f(y)}" width="${f(w)}" height="${f(h)}" rx="${f(rx)}" fill="${fill}"/></g>`;
  const front = `<g transform="rotate(7 ${f(512)} ${f(540)})"><rect x="${f(350)}" y="${f(236)}" width="${f(340)}" height="${f(520)}" rx="${f(42)}" fill="${col.front}"/><path d="M${f(468)} ${f(404)} V${f(588)} L${f(630)} ${f(496)} Z" fill="${col.pl}" stroke="${col.pl}" stroke-width="${f(30)}" stroke-linejoin="round"/></g>`;
  return `<svg width="${size}" height="${size}" viewBox="0 0 ${size} ${size}" fill="none" xmlns="http://www.w3.org/2000/svg"><defs><linearGradient id="g${size}${mode}" x1="0" y1="0" x2="0" y2="${size}" gradientUnits="userSpaceOnUse"><stop stop-color="${col.g[0]}"/><stop offset="1" stop-color="${col.g[1]}"/></linearGradient></defs><rect width="${size}" height="${size}" fill="url(#g${size}${mode})"/>${rr(-16, 512, 540, 340, 250, 330, 500, 40, col.back)}${rr(-6, 512, 540, 350, 240, 330, 500, 40, col.mid)}${front}</svg>`;
};
C.items = [{"t":"Dune: Part Two","y":"2024","k":"movie","r":8.2,"s":"available"},{"t":"Oppenheimer","y":"2023","k":"movie","r":8.1,"s":"requested"},{"t":"The Bear","y":"2022","k":"tv","r":8.6,"s":"approved"},{"t":"Severance","y":"2022","k":"tv","r":8.7,"s":"downloading"},{"t":"Past Lives","y":"2023","k":"movie","r":7.9,"s":null},{"t":"Shōgun","y":"2024","k":"tv","r":8.6,"s":"partial"},{"t":"Poor Things","y":"2023","k":"movie","r":7.8,"s":null},{"t":"Andor","y":"2022","k":"tv","r":8.4,"s":"searching"},{"t":"Anatomy of a Fall","y":"2023","k":"movie","r":7.7,"s":"declined"},{"t":"Silo","y":"2023","k":"tv","r":8.1,"s":null},{"t":"Challengers","y":"2024","k":"movie","r":7.1,"s":"failed"},{"t":"Fallout","y":"2024","k":"tv","r":8.3,"s":null}];
C.largeTitle = (p, T, title, o = {}) => {
  H.text(p, title, 16, o.y ?? 66, { size: 34, weight: 700, color: T.fg, name: 'large title', ls: -0.4 });
  (o.right || []).forEach((ic, i) => { const bt = H.board(p, 'nav button ' + ic, 393 - 16 - 44 * (i + 1) - 8 * i, (o.y ?? 66) - 2, 44, 44, { fill: T.muted, r: 22, clip: false }); H.icon(bt, ic, 11, 11, 22, T.fg, 2); });
};
C.pal = [["#3B4A6B","#0F1626"],["#6B3B3B","#1A0E0E"],["#2F5D50","#0C1A16"],["#5B4A8A","#14102A"],["#8A6B3B","#1F150A"],["#3B6B7A","#0B1A20"],["#7A3B5B","#220F18"],["#4A5B3B","#121A0C"],["#6B5B4A","#1A140F"],["#3B3B6B","#0F0F26"],["#7A4A3B","#200F0A"],["#3B7A6B","#0A1F1A"]];
C.personRow = (p, T, x, y, w, name, known, ci) => {
  const pal = C.pal[ci % C.pal.length];
  H.ellipse(p, 'avatar', x, y, 56, 56, [H.grad([[pal[0], 0], [pal[1], 1]])]);
  H.text(p, name.split(' ').map(s => s[0]).join(''), x, y + 17, { size: 18, weight: 600, color: '#FFFFFF', w: 56, align: 'center', name: 'initials' });
  H.text(p, name, x + 72, y + 8, { size: 16, weight: 600, color: T.fg, w: w - 72 });
  H.text(p, known, x + 72, y + 30, { size: 13, color: T.mutedFg, w: w - 72 });
};
C.poster = (p, T, x, y, w, i, o = {}) => {
  const it = C.items[i % C.items.length]; const h = Math.round(w * 1.5); const pal = C.pal[i % C.pal.length];
  const b = H.board(p, 'poster ' + it.t, x, y, w, h, { fill: [H.grad([[pal[0], 0], [pal[1], 1]])], r: o.r ?? 10, clip: true, stroke: T.isDark ? ['#FFFFFF', 1, 0.08] : null });
  if (!o.noIcon) H.icon(b, it.k === 'tv' ? 'tv' : 'film', w / 2 - 14, h / 2 - 22, Math.min(28, w * 0.4), '#FFFFFF', 1.5).opacity = 0.35;
  if (!o.noTitle) H.text(b, it.t, 8, h - (w > 110 ? 40 : 32), { size: w > 110 ? 13 : 11, weight: 700, color: '#FFFFFF', w: w - 16, lh: 1.15, name: 'title' });
  if (o.badge !== false && it.s) C.badge(b, T, w - 8 - 22, 8, it.s, { iconOnly: true, solid: true, h: 22 });
  if (o.rating) { const rb = H.board(b, 'rating', 8, 8, 42, 20, { fill: '#000000', op: 0.55, r: 10, clip: false }); H.icon(rb, 'star', 6, 4, 12, '#F59E0B', 2, true); H.text(rb, it.r.toFixed(1), 20, 3, { size: 11, weight: 600, color: '#FFFFFF' }); }
  return b;
};
C.resultRow = (p, T, x, y, w, i, o = {}) => {
  const it = C.items[i % C.items.length];
  C.poster(p, T, x, y, 64, i, { noTitle: true, noIcon: false, badge: false, r: 8 });
  H.text(p, it.t, x + 80, y + 2, { size: 16, weight: 600, color: T.fg, w: w - 80, name: 'result title' });
  H.text(p, `${it.y}  ·  ${it.k === 'tv' ? 'TV show' : 'Movie'}  ·  ★ ${it.r.toFixed(1)}`, x + 80, y + 24, { size: 13, color: T.mutedFg, w: w - 80, name: 'meta' });
  H.text(p, o.overview || 'A short synopsis of the title sits here and wraps onto two lines at most before it is cut.', x + 80, y + 44, { size: 13, color: T.mutedFg, w: w - 80, lh: 1.3, name: 'overview', op: 0.9 });
  if (it.s) C.badge(p, T, x + 80, y + 78, it.s, {});
  else C.btn(p, T, x + 80, y + 76, 92, 'Request', { h: 28, size: 13, v: 'secondary', icon: 'plus' });
};
C.screen = (T, name, x, y, o = {}) => {
  const b = H.board(null, name, x, y, 393, 852, { fill: T.bg, r: 0 });
  if (o.status !== false) C.statusBar(b, T);
  return b;
};
C.search = (p, T, x, y, w, placeholder = 'Movies, shows, people…', o = {}) => {
  const b = H.board(p, 'search field', x, y, w, 44, { fill: T.muted, r: 22, clip: false });
  H.icon(b, 'search', 14, 12, 20, T.mutedFg, 2);
  H.text(b, o.value || placeholder, 44, 11.5, { size: 17, color: o.value ? T.fg : T.mutedFg, name: 'placeholder' });
  if (o.value) { H.ellipse(b, 'clear bg', w - 34, 12, 20, 20, T.mutedFg, { op: 0.5 }); H.icon(b, 'x', w - 31, 15, 14, T.bg, 2.6); }
  return b;
};
C.sectionHeader = (p, T, x, y, w, title, action = 'See all') => {
  H.text(p, title, x, y, { size: 20, weight: 700, color: T.fg, name: 'section ' + title });
  if (action) { H.text(p, action, x, y + 3, { size: 15, weight: 500, color: T.mutedFg, w, align: 'right', name: 'action' }); }
};
C.segmented = (p, T, x, y, w, labels, active) => {
  const b = H.board(p, 'segmented', x, y, w, 36, { fill: T.muted, r: 18, clip: false });
  const sw = (w - 4) / labels.length;
  labels.forEach((l, i) => { if (i === active) H.rect(b, 'selected', 2 + i * sw, 2, sw, 32, T.isDark ? '#3A3A3A' : '#FFFFFF', { r: 16, stroke: T.isDark ? null : [T.border, 1] }); H.text(b, l, 2 + i * sw, 8, { size: 14, weight: i === active ? 600 : 500, color: i === active ? T.fg : T.mutedFg, w: sw, align: 'center', name: 'seg ' + l }); });
  return b;
};
C.status = {"available":{"c":"#10B981","label":"Available","icon":"check"},"partial":{"c":"#14B8A6","label":"Partial","icon":"circle-dashed"},"requested":{"c":"#F59E0B","label":"Requested","icon":"clock"},"approved":{"c":"#0EA5E9","label":"Approved","icon":"thumbs-up"},"searching":{"c":"#D946EF","label":"Searching","icon":"search"},"downloading":{"c":"#6366F1","label":"Downloading","icon":"download"},"declined":{"c":"#F43F5E","label":"Declined","icon":"ban"},"failed":{"c":"#EA580C","label":"Failed","icon":"triangle-alert"}};
C.statusBar = (p, T) => {
  H.text(p, '9:41', 0, 19, { size: 17, weight: 600, color: T.fg, w: 134, align: 'center', name: 'time' });
  H.rect(p, 'dynamic island', 133.5, 11, 126, 37, '#000000', { r: 19 });
  const bx = 393 - 28 - 25; // battery
  H.rect(p, 'battery', bx, 24, 25, 12, null, { r: 4, stroke: [T.fg, 1, 0.4] });
  H.rect(p, 'battery level', bx + 2, 26, 21, 8, T.fg, { r: 2.5 });
  for (let i = 0; i < 4; i++) H.rect(p, 'signal', 393 - 28 - 25 - 14 - 30 + i * 5, 32 - (i + 1) * 3.2, 3, (i + 1) * 3.2, T.fg, { r: 1 });
};
C.tabBar = (p, T, active) => {
  const tabs = [['compass', 'Discover'], ['search', 'Search'], ['list-checks', 'Requests'], ['library', 'Library'], ['user', 'Profile']];
  const bar = H.board(p, 'tab bar', 16, 852 - 34 - 64, 361, 64, { fill: T.card, r: 32, stroke: [T.border, 1], clip: false });
  const w = 361 / 5;
  tabs.forEach(([ic, label], i) => {
    const on = i === active;
    if (on) H.rect(bar, 'active pill', i * w + 4, 6, w - 8, 52, T.muted, { r: 26 });
    H.icon(bar, ic, i * w + (w - 24) / 2, 10, 24, on ? T.fg : T.mutedFg, on ? 2.2 : 1.8);
    H.text(bar, label, i * w, 38, { size: 10, weight: on ? 600 : 500, color: on ? T.fg : T.mutedFg, w, align: 'center', name: 'tab ' + label });
  });
  return bar;
};
C.themes = {"light":{"name":"light","bg":"#FFFFFF","fg":"#0A0A0A","card":"#FFFFFF","muted":"#F5F5F5","mutedFg":"#737373","border":"#E5E5E5","primary":"#171717","primaryFg":"#FAFAFA","destructive":"#E7000B","ring":"#A3A3A3","overlay":"#000000","tabBar":"#FFFFFF","shadow":0.1,"isDark":false},"dark":{"name":"dark","bg":"#0A0A0A","fg":"#FAFAFA","card":"#171717","muted":"#262626","mutedFg":"#A3A3A3","border":"#2A2A2A","primary":"#E5E5E5","primaryFg":"#171717","destructive":"#FF6467","ring":"#737373","overlay":"#000000","tabBar":"#171717","shadow":0.4,"isDark":true}};
H._fonts = {};
H.asFill = (f, op = 1) => f == null ? [] : (typeof f === 'string' ? H.fill(f, op) : f);
H.board = (parent, name, x, y, w, h, o = {}) => { const b = penpot.createBoard(); b.name = name; b.resize(w, h); if (parent) parent.appendChild(b); H.pos(b, parent, x, y); b.fills = H.asFill(o.fill, o.op ?? 1); if (o.r) b.borderRadius = o.r; if (o.stroke) b.strokes = H.stroke(...o.stroke); try { b.clipContent = o.clip !== false; } catch (e) {} return b; };
H.ellipse = (parent, name, x, y, w, h, fill, o = {}) => { const r = penpot.createEllipse(); r.name = name; r.resize(w, h); parent.appendChild(r); H.pos(r, parent, x, y); r.fills = H.asFill(fill, o.op ?? 1); if (o.stroke) r.strokes = H.stroke(...o.stroke); return r; };
H.fill = (c, o = 1) => [{ fillColor: c, fillOpacity: o }];
H.font = (name) => H._fonts[name] || (H._fonts[name] = penpot.fonts.all.find(f => f.name === name));
H.grad = (stops, dir = 'v') => ({ fillColorGradient: { type: 'linear', startX: dir === 'h' ? 0 : 0.5, startY: dir === 'h' ? 0.5 : 0, endX: dir === 'h' ? 1 : 0.5, endY: dir === 'h' ? 0.5 : 1, width: 1, stops: stops.map(([c, o, op]) => ({ color: c, offset: o, opacity: op ?? 1 })) } });
H.icon = (parent, name, x, y, size, color, sw = 2, filled = false) => {
  const inner = storage.icons[name].replace(/currentColor/g, color);
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="${filled ? color : 'none'}" stroke="${color}" stroke-width="${sw}" stroke-linecap="round" stroke-linejoin="round">${inner}</svg>`;
  const g = penpot.createShapeFromSvg(svg); g.name = 'icon/' + name;
  for (const k of g.children) if (k.name === 'base-background') { k.strokes = []; k.fills = []; k.name = 'icon box'; }
  parent.appendChild(g);
  if (size !== 24) g.resize(size, size);
  const k = size / 24; const walk = (s) => { if (s.name !== 'icon box' && s.strokes && s.strokes.length) s.strokes = s.strokes.map(st => ({ ...st, strokeWidth: sw * k })); (s.children || []).forEach(walk); }; walk(g);
  H.pos(g, parent, x, y); return g;
};
H.pos = (s, parent, x, y) => { s.x = (parent ? parent.x : 0) + x; s.y = (parent ? parent.y : 0) + y; };
H.rect = (parent, name, x, y, w, h, fill, o = {}) => { const r = penpot.createRectangle(); r.name = name; r.resize(w, h); parent.appendChild(r); H.pos(r, parent, x, y); r.fills = H.asFill(fill, o.op ?? 1); if (o.r != null) r.borderRadius = o.r; if (o.stroke) r.strokes = H.stroke(...o.stroke); if (o.opacity != null) r.opacity = o.opacity; return r; };
H.snap = async (shape, name, wait = 1200) => { await new Promise(r => setTimeout(r, wait)); const m = penpot.generateMarkup([shape], { type: 'svg' }); await fetch('http://127.0.0.1:8765/' + name, { method: 'POST', body: m }); return m.length; };
H.stroke = (c, w = 1, o = 1, al = 'inner') => [{ strokeColor: c, strokeWidth: w, strokeOpacity: o, strokeAlignment: al, strokeStyle: 'solid' }];
H.text = (parent, str, x, y, o = {}) => {
  const t = penpot.createText(String(str) === '' ? ' ' : String(str)); parent.appendChild(t); t.name = o.name || String(str).slice(0, 30).trim() || 'text';
  const f = H.font(o.family || 'Inter'); const w = String(o.weight || 400);
  const v = f.variants.find(v => v.fontWeight === w && v.fontStyle === (o.italic ? 'italic' : 'normal')) || f.variants[0];
  f.applyToText(t, v);
  t.fontSize = String(o.size || 16); t.fills = H.fill(o.color || '#000000', o.op ?? 1); t.align = o.align || 'left';
  if (o.lh) t.lineHeight = String(o.lh); if (o.ls != null) t.letterSpacing = String(o.ls); if (o.upper) t.textTransform = 'uppercase';
  if (o.w) { t.resize(o.w, Math.max(10, (o.size || 16) * 1.3)); t.growType = 'auto-height'; } else t.growType = 'auto-width';
  H.pos(t, parent, x, y); return t;
};
S.discoverA = (T, x, y) => {
  const b = C.screen(T, `Discover A · hero + rails · ${T.name}`, x, y, { status: false });
  const pal = C.pal[0];
  const hero = H.board(b, 'hero', 0, 0, 393, 500, { fill: [H.grad([[pal[0], 0], [pal[1], 0.7], [T.bg, 0.93]])], clip: true });
  H.icon(hero, 'film', 160, 140, 72, '#FFFFFF', 1.2).opacity = 0.18;
  C.statusBar(b, { ...T, fg: '#FFFFFF' });
  H.text(b, 'Dune: Part Two', 16, 300, { size: 34, weight: 700, color: '#FFFFFF', name: 'hero title', ls: -0.4 });
  H.text(b, '2024  ·  Movie  ·  2h 46m  ·  Sci-Fi', 16, 344, { size: 14, weight: 500, color: '#D4D4D8', name: 'hero meta' });
  const r = H.board(b, 'rating', 16, 380, 52, 26, { fill: T.muted, r: 13, clip: false }); H.icon(r, 'star', 8, 6, 14, '#F59E0B', 2, true); H.text(r, '8.2', 26, 5, { size: 13, weight: 600, color: T.fg });
  C.badge(b, T, 76, 382, 'available', {});
  C.btn(b, T, 16, 424, 290, 'Request', { icon: 'plus', h: 46 });
  const info = H.board(b, 'info button', 316, 424, 46, 46, { fill: T.muted, r: 23, clip: false }); H.icon(info, 'info', 12, 12, 22, T.fg, 2);
  [0, 1, 2].forEach(i => H.ellipse(b, 'page dot', 176 + i * 14, 484, i === 0 ? 8 : 6, i === 0 ? 8 : 6, i === 0 ? T.fg : T.mutedFg, { op: i === 0 ? 1 : 0.5 }));
  C.sectionHeader(b, T, 16, 510, 361, 'Trending now');
  [1, 2, 3, 4].forEach((n, i) => C.poster(b, T, 16 + i * 126, 546, 110, n));
  C.sectionHeader(b, T, 16, 732, 361, 'Upcoming');
  [5, 6, 7, 8].forEach((n, i) => C.poster(b, T, 16 + i * 126, 768, 110, n));
  C.finish(b, T, { tab: 0 });
  return b;
};
S.discoverB = (T, x, y) => {
  const b = C.screen(T, `Discover B · grid + chips · ${T.name}`, x, y);
  C.largeTitle(b, T, 'Discover', { y: 62, right: ['search'] });
  let cx = 16;
  [['Trending', true, 'trending-up'], ['Upcoming', false, 'calendar'], ['Movies', false, 'film'], ['TV', false, 'tv'], ['Box office', false, 'popcorn']].forEach(([l, a, ic]) => { const c = C.chip(b, T, cx, 120, l, { active: a, icon: ic }); cx += c.width + 8; });
  C.sectionHeader(b, T, 16, 176, 361, 'Trending this week', 'See all');
  const w = 112;
  for (let i = 0; i < 3; i++) { const px = 16 + i * (w + 9), py = 212; C.poster(b, T, px, py, w, i);
    const it = C.items[i]; H.text(b, it.t, px, py + 172, { size: 13, weight: 600, color: T.fg, w, name: 'poster title' });
    H.text(b, `${it.y} · ${it.k === 'tv' ? 'TV' : 'Movie'}`, px, py + 190, { size: 12, color: T.mutedFg, w }); }
  C.sectionHeader(b, T, 16, 432, 361, 'Suggested for you', 'See all');
  H.text(b, 'Because you watched Severance', 16, 460, { size: 13, color: T.mutedFg, name: 'reason' });
  for (let i = 0; i < 3; i++) { const px = 16 + i * (w + 9), py = 488; C.poster(b, T, px, py, w, i + 3);
    const it = C.items[i + 3]; H.text(b, it.t, px, py + 172, { size: 13, weight: 600, color: T.fg, w });
    H.text(b, `${it.y} · ${it.k === 'tv' ? 'TV' : 'Movie'}`, px, py + 190, { size: 12, color: T.mutedFg, w }); }
  C.finish(b, T, { tab: 0 });
  return b;
};
S.discoverError = (T, x, y) => {
  const b = C.screen(T, `Discover · TMDB not configured · ${T.name}`, x, y);
  C.largeTitle(b, T, 'Discover', { y: 62 });
  S.empty(b, T, { y: 260, icon: 'server', title: 'Discover isn’t set up yet', body: 'The server has no TMDB key. Ask an admin to add it in Settings so trending and search work.', cta: 'Retry', ctaIcon: 'refresh-cw', ctaW: 140, ctaV: 'secondary' });
  C.finish(b, T, { tab: 0 });
  return b;
};
S.discoverLoading = (T, x, y) => {
  const b = C.screen(T, `Discover · loading · ${T.name}`, x, y);
  C.largeTitle(b, T, 'Discover', { y: 62 });
  let cx = 16; [78, 82, 70, 48, 90].forEach(w => { S.skel(b, T, cx, 120, w, 34, 17); cx += w + 8; });
  S.skel(b, T, 16, 180, 160, 22, 6);
  for (let i = 0; i < 3; i++) { S.skel(b, T, 16 + i * 121, 212, 112, 168, 10); S.skel(b, T, 16 + i * 121, 388, 96, 12, 4); S.skel(b, T, 16 + i * 121, 406, 60, 10, 4); }
  S.skel(b, T, 16, 440, 180, 22, 6);
  for (let i = 0; i < 3; i++) S.skel(b, T, 16 + i * 121, 480, 112, 168, 10);
  C.finish(b, T, { tab: 0 });
  return b;
};
S.discoverOffline = (T, x, y) => {
  const b = C.screen(T, `Discover · offline · ${T.name}`, x, y);
  C.largeTitle(b, T, 'Discover', { y: 62 });
  S.empty(b, T, { y: 260, icon: 'wifi-off', title: 'No connection', body: 'Tipsarr can’t reach your server. Check your internet connection and try again.', cta: 'Try again', ctaIcon: 'refresh-cw', ctaW: 160 });
  C.finish(b, T, { tab: 0 });
  return b;
};
S.dump = async (tag = 'lib') => { const dump = {}; for (const [ns, o] of Object.entries({ H: storage.H, C: storage.C, S: storage.S })) for (const [k, v] of Object.entries(o)) dump[`${ns}.${k}`] = typeof v === 'function' ? v.toString() : JSON.stringify(v); await fetch('http://127.0.0.1:8765/' + tag, { method: 'POST', body: JSON.stringify(dump) }); return Object.keys(dump).length; };
S.empty = (b, T, o) => {
  const cy = o.y ?? 300;
  H.ellipse(b, 'empty icon bg', 393 / 2 - 36, cy, 72, 72, T.muted);
  H.icon(b, o.icon, 393 / 2 - 18, cy + 18, 36, o.iconColor || T.mutedFg, 1.8);
  H.text(b, o.title, 40, cy + 92, { size: 20, weight: 700, color: T.fg, w: 313, align: 'center', name: 'empty title' });
  H.text(b, o.body, 40, cy + 124, { size: 15, color: T.mutedFg, w: 313, align: 'center', lh: 1.35, name: 'empty body' });
  const by = cy + 124 + (o.bodyLines || 3) * 21 + 22;
  if (o.cta) C.btn(b, T, 393 / 2 - (o.ctaW || 200) / 2, by, o.ctaW || 200, o.cta, { icon: o.ctaIcon, h: 46, v: o.ctaV || 'primary' });
  if (o.cta2) C.btn(b, T, 393 / 2 - 100, by + 56, 200, o.cta2, { h: 46, v: 'ghost' });
};
S.label = (txt, x, y, color = '#9CA3AF') => H.text(penpot.root, txt, x, y - 36, { size: 20, weight: 600, color, name: 'label ' + txt });
S.layout = (list, cols = 6) => { const ids = {}; [C.themes.light, C.themes.dark].forEach((T, ri) => list.forEach(([label, fn], i) => { const col = i % cols, row = Math.floor(i / cols) + ri * Math.ceil(list.length / cols); const x = col * 460, y = row * 1050; const b = fn(T, x, y); S.label(`${label} · ${T.name}`, x, y); ids[`${i}_${T.name}`] = b.id; })); return ids; };
S.skel = (p, T, x, y, w, h, r = 8) => H.rect(p, 'skeleton', x, y, w, h, T.muted, { r });
S.snapAll = async (ids, prefix = '', wait = 1800) => { await new Promise(r => setTimeout(r, wait)); const out = {}; for (const [k, id] of Object.entries(ids)) { const sh = penpotUtils.findShapeById(id); const m = penpot.generateMarkup([sh], { type: 'svg' }); await fetch('http://127.0.0.1:8765/' + prefix + k, { method: 'POST', body: m }); out[k] = m.length; } return out; };
