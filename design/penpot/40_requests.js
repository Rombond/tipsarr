const H = storage.H, C = storage.C, S = storage.S;
C.avatarSm = (p, T, x, y, name, ci, size = 22) => { const pal = C.pal[ci % C.pal.length]; H.ellipse(p, 'avatar ' + name, x, y, size, size, [H.grad([[pal[0], 0], [pal[1], 1]])]); H.text(p, name[0], x, y + size * 0.22, { size: size * 0.5, weight: 600, color: '#FFFFFF', w: size, align: 'center', name: 'initial' }); };
C.requestRow = (p, T, x, y, w, i, o = {}) => {
  const it = C.items[i % C.items.length]; const st = o.status || it.s || 'requested';
  C.poster(p, T, x, y, 64, i, { badge: false, noTitle: true, r: 8 });
  H.text(p, it.t, x + 80, y + 2, { size: 16, weight: 600, color: T.fg, w: w - 80 - (o.admin ? 0 : 0), name: 'req title' });
  H.text(p, `${it.y}  ·  ${it.k === 'tv' ? 'TV show' : 'Movie'}`, x + 80, y + 24, { size: 13, color: T.mutedFg, w: w - 80 });
  if (o.admin) { C.avatarSm(p, T, x + 80, y + 46, o.who || 'Bob', o.ci ?? 2, 20); H.text(p, `${o.who || 'Bob'}  ·  ${o.when || '2h ago'}`, x + 106, y + 48, { size: 12, color: T.mutedFg }); }
  else H.text(p, `Requested ${o.when || '2 days ago'}`, x + 80, y + 46, { size: 12, color: T.mutedFg });
  if (st === 'downloading') { C.progress(p, T, x + 80, y + 80, w - 80 - 8, o.pct ?? 0.62); }
  const bd = C.badge(p, T, x + w - 8, y + 2, st, {}); bd.x = x + w - bd.width; 
  if (o.actions) { C.btn(p, T, x + 80, y + 72, 98, 'Approve', { h: 32, size: 14, icon: 'check' }); C.btn(p, T, x + 186, y + 72, 98, 'Decline', { h: 32, size: 14, v: 'secondary', icon: 'x' }); }
};
C.timeline = (p, T, x, y, steps, cur, o = {}) => {
  steps.forEach(([label, time], i) => {
    const yy = y + i * 56; const done = i < cur, on = i === cur, bad = o.failedAt === i;
    if (i < steps.length - 1) H.rect(p, 'line', x + 11, yy + 26, 2, 34, done ? '#10B981' : T.border, {});
    if (bad) { H.ellipse(p, 'step failed', x, yy, 24, 24, '#EA580C'); H.icon(p, 'x', x + 5, yy + 5, 14, '#FFFFFF', 3); }
    else if (done) { H.ellipse(p, 'step done', x, yy, 24, 24, '#10B981'); H.icon(p, 'check', x + 5, yy + 5, 14, '#FFFFFF', 3); }
    else if (on) { H.ellipse(p, 'step current', x, yy, 24, 24, '#6366F1', { op: 0.2 }); H.ellipse(p, 'step current dot', x + 7, yy + 7, 10, 10, '#6366F1'); }
    else H.ellipse(p, 'step todo', x, yy, 24, 24, null, { stroke: [T.border, 2] });
    H.text(p, label, x + 40, yy + 1, { size: 16, weight: on || bad ? 700 : 500, color: done || on || bad ? T.fg : T.mutedFg });
    H.text(p, time, x + 40, yy + 22, { size: 12, color: T.mutedFg });
  });
};
const tabs = (b, T, active) => { let cx = 16; ['All', 'Pending', 'Approved', 'Available', 'Declined'].forEach((l, i) => { const c = C.chip(b, T, cx, 118, l, { active: i === active }); cx += c.width + 8; }); };
const detailTop = (b, T, i, title) => {
  S.navBar(b, T, title);
  C.poster(b, T, 16, 112, 96, i, { badge: false, noTitle: true, r: 10 });
  const it = C.items[i]; H.text(b, it.t, 128, 114, { size: 20, weight: 700, color: T.fg, w: 249, lh: 1.15 }); H.text(b, `${it.y}  ·  ${it.k === 'tv' ? 'TV show' : 'Movie'}`, 128, 168, { size: 13, color: T.mutedFg });
};
const facts = (b, T, y, rows) => { rows.forEach(([k, v], j) => { H.text(b, k, 16, y + j * 34, { size: 15, color: T.mutedFg }); H.text(b, v, 16, y + j * 34, { size: 15, weight: 500, color: T.fg, w: 361, align: 'right' }); H.rect(b, 'divider', 16, y + j * 34 + 26, 361, 1, T.border); }); };
const st5 = [['Requested', 'Mar 3, 18:02'], ['Approved', 'Mar 3, 18:05'], ['Searching', 'Mar 3, 18:05'], ['Downloading', 'in progress'], ['Available', 'waiting']];
S.requestsList = [
  ['Requests · mine', (T, x, y) => { const b = C.screen(T, `Requests · mine · ${T.name}`, x, y); C.largeTitle(b, T, 'Requests', { y: 62 }); tabs(b, T, 0);
    [[1, 'requested', '2 days ago'], [3, 'downloading', '3 days ago'], [0, 'available', '1 week ago'], [8, 'declined', '2 weeks ago'], [10, 'failed', '3 weeks ago']].forEach(([i, s, w], k) => { C.requestRow(b, T, 16, 176 + k * 116, 361, i, { status: s, when: w }); H.rect(b, 'divider', 16, 176 + k * 116 + 100, 361, 1, T.border); });
    C.finish(b, T, { tab: 2 }); return b; }],
  ['Requests · admin', (T, x, y) => { const b = C.screen(T, `Requests · admin pending · ${T.name}`, x, y); C.largeTitle(b, T, 'Requests', { y: 62 }); tabs(b, T, 1);
    [[1, 'Alice', 0], [5, 'Bob', 2], [7, 'Carol', 4], [9, 'Dan', 6]].forEach(([i, who, ci], k) => { C.requestRow(b, T, 16, 176 + k * 128, 361, i, { status: 'requested', who, ci, when: '2h ago', admin: true, actions: true }); H.rect(b, 'divider', 16, 176 + k * 128 + 112, 361, 1, T.border); });
    C.finish(b, T, { tab: 2 }); return b; }],
  ['Requests · empty', (T, x, y) => { const b = C.screen(T, `Requests · empty · ${T.name}`, x, y); C.largeTitle(b, T, 'Requests', { y: 62 }); S.empty(b, T, { y: 250, icon: 'list-checks', title: 'No requests yet', body: 'Find something to watch in Discover and request it. You’ll follow its progress here.', cta: 'Browse Discover', ctaIcon: 'compass', ctaW: 210, bodyLines: 2 }); C.finish(b, T, { tab: 2 }); return b; }],
  ['Requests · loading', (T, x, y) => { const b = C.screen(T, `Requests · loading · ${T.name}`, x, y); C.largeTitle(b, T, 'Requests', { y: 62 }); let cx = 16; [40, 76, 88, 84, 78].forEach(w => { S.skel(b, T, cx, 118, w + 8, 34, 17); cx += w + 16; });
    for (let k = 0; k < 5; k++) { const yy = 176 + k * 116; S.skel(b, T, 16, yy, 64, 96, 8); S.skel(b, T, 96, yy + 4, 180, 16, 5); S.skel(b, T, 96, yy + 30, 110, 12, 4); S.skel(b, T, 96, yy + 52, 140, 12, 4); } C.finish(b, T, { tab: 2 }); return b; }],
  ['Requests · swipe', (T, x, y) => { const b = C.screen(T, `Requests · swipe to cancel · ${T.name}`, x, y); C.largeTitle(b, T, 'Requests', { y: 62 }); tabs(b, T, 0);
    [[1, 'requested', '2 days ago'], [3, 'downloading', '3 days ago'], [0, 'available', '1 week ago']].forEach(([i, s, w], k) => { const sw = k === 0; const row = H.board(b, 'row clip', 0, 176 + k * 116, 393, 100, { fill: sw ? '#EF4444' : null, clip: true }); if (sw) { H.icon(row, 'trash-2', 393 - 54, 34, 26, '#FFFFFF', 2); H.text(row, 'Cancel', 393 - 78, 64, { size: 12, weight: 600, color: '#FFFFFF', w: 62, align: 'center' }); } const inner = H.board(row, 'row content', sw ? -90 : 0, 0, 393, 100, { fill: T.bg, clip: false }); C.requestRow(inner, T, 16, 0, 361, i, { status: s, when: w }); });
    C.finish(b, T, { tab: 2 }); return b; }],
  ['Request detail · downloading', (T, x, y) => { const b = C.screen(T, `Request detail · downloading · ${T.name}`, x, y); detailTop(b, T, 3, 'Request'); C.badge(b, T, 128, 196, 'downloading', {}); C.progress(b, T, 16, 272, 361, 0.62); H.text(b, '62%  ·  about 12 min left', 16, 286, { size: 13, color: T.mutedFg });
    H.text(b, 'Progress', 16, 322, { size: 20, weight: 700, color: T.fg }); C.timeline(b, T, 20, 366, st5, 3);
    H.text(b, 'Details', 16, 654, { size: 20, weight: 700, color: T.fg }); facts(b, T, 692, [['Requested by', 'You'], ['Quality profile', 'HD - 1080p']]);
    C.btn(b, T, 16, 770, 361, 'Cancel request', { h: 46, v: 'secondary' }); C.homeIndicator(b, T); return b; }],
  ['Request detail · pending (admin)', (T, x, y) => { const b = C.screen(T, `Request detail · pending admin · ${T.name}`, x, y); detailTop(b, T, 1, 'Request'); C.badge(b, T, 128, 196, 'requested', {});
    H.text(b, 'Requested by', 16, 262, { size: 13, weight: 600, color: T.mutedFg, upper: true, ls: 0.4 }); C.avatarSm(b, T, 16, 288, 'Bob', 2, 36); H.text(b, 'Bob', 62, 288, { size: 16, weight: 600, color: T.fg }); H.text(b, '2 hours ago', 62, 308, { size: 13, color: T.mutedFg });
    H.text(b, 'Progress', 16, 366, { size: 20, weight: 700, color: T.fg }); C.timeline(b, T, 20, 410, [['Requested', 'today, 16:02'], ['Approved', 'waiting'], ['Searching', ''], ['Downloading', ''], ['Available', '']], 0);
    H.text(b, 'Details', 16, 700, { size: 20, weight: 700, color: T.fg }); facts(b, T, 738, [['Quality profile', 'HD - 1080p']]);
    C.btn(b, T, 16, 770, 176, 'Approve', { h: 50, icon: 'check' }); C.btn(b, T, 201, 770, 176, 'Decline', { h: 50, v: 'secondary', icon: 'x' }); C.homeIndicator(b, T); return b; }],
  ['Request detail · declined', (T, x, y) => { const b = C.screen(T, `Request detail · declined · ${T.name}`, x, y); detailTop(b, T, 8, 'Request'); C.badge(b, T, 128, 196, 'declined', {});
    C.banner(b, T, 16, 272, 361, 'error', 'Declined by Alice: we already have a better version coming.');
    H.text(b, 'Progress', 16, 366, { size: 20, weight: 700, color: T.fg }); C.timeline(b, T, 20, 410, [['Requested', 'Feb 20, 11:40'], ['Declined', 'Feb 21, 09:12']], 1, { failedAt: 1 });
    C.btn(b, T, 16, 770, 361, 'Request again', { h: 50, icon: 'rotate-ccw' }); C.homeIndicator(b, T); return b; }],
  ['Request detail · failed (admin)', (T, x, y) => { const b = C.screen(T, `Request detail · failed admin · ${T.name}`, x, y); detailTop(b, T, 10, 'Request'); C.badge(b, T, 128, 196, 'failed', {});
    C.banner(b, T, 16, 272, 361, 'warn', 'Radarr could not find a release for this title after 3 tries.');
    H.text(b, 'Progress', 16, 366, { size: 20, weight: 700, color: T.fg }); C.timeline(b, T, 20, 410, [['Requested', 'Mar 1, 20:15'], ['Approved', 'Mar 1, 20:15'], ['Searching', 'Mar 1, 20:16'], ['Failed', 'Mar 2, 08:00']], 3, { failedAt: 3 });
    C.btn(b, T, 16, 770, 361, 'Retry', { h: 50, icon: 'refresh-cw' }); C.homeIndicator(b, T); return b; }],
  ['Request detail · available', (T, x, y) => { const b = C.screen(T, `Request detail · available · ${T.name}`, x, y); detailTop(b, T, 0, 'Request'); C.badge(b, T, 128, 196, 'available', {});
    H.text(b, 'Progress', 16, 276, { size: 20, weight: 700, color: T.fg }); C.timeline(b, T, 20, 320, [['Requested', 'Feb 20, 11:40'], ['Approved', 'Feb 20, 11:41'], ['Searching', 'Feb 20, 11:41'], ['Downloading', 'Feb 20, 11:43'], ['Available', 'Feb 20, 12:20']], 5);
    C.btn(b, T, 16, 700, 361, 'Open in Jellyfin', { h: 50, icon: 'play', v: 'success' }); C.btn(b, T, 16, 760, 361, 'View details', { h: 46, v: 'secondary' }); C.homeIndicator(b, T); return b; }],
  ['Decline sheet (admin)', (T, x, y) => { const b = C.screen(T, `Decline sheet · ${T.name}`, x, y); detailTop(b, T, 1, 'Request'); const s = C.sheet(b, T, 470, 'Decline request');
    H.text(s, 'Optionally tell Bob why.', 20, 62, { size: 15, color: T.mutedFg }); const ta = H.board(s, 'textarea', 20, 100, 353, 130, { fill: T.muted, r: 14, clip: false }); H.text(ta, 'Reason (optional)…', 16, 14, { size: 16, color: T.mutedFg });
    let cx = 20; ['Already available', 'Not suitable', 'Duplicate'].forEach(l => { const c = C.chip(s, T, cx, 246, l, {}); cx += c.width + 8; });
    C.btn(s, T, 20, 470 - 34 - 52 - 52 - 6, 353, 'Decline', { h: 52, v: 'destructive' }); C.btn(s, T, 20, 470 - 34 - 52, 353, 'Cancel', { h: 46, v: 'ghost' }); return b; }],
  ['Requests · filter menu', (T, x, y) => { const b = C.screen(T, `Requests · filter · ${T.name}`, x, y); C.largeTitle(b, T, 'Requests', { y: 62, right: ['sliders-horizontal'] }); tabs(b, T, 0); const s = C.sheet(b, T, 520, 'Filter');
    C.group(s, T, 20, 76, 353, [{ title: 'All requests', icon: 'list', chev: false }, { title: 'Mine', icon: 'user', chev: false }, { title: 'Pending approval', icon: 'clock', chev: false }, { title: 'Approved', icon: 'thumbs-up', chev: false }, { title: 'Available', icon: 'check', chev: false }, { title: 'Declined', icon: 'ban', chev: false }, { title: 'Failed', icon: 'triangle-alert', chev: false }]);
    H.icon(s, 'check', 393 - 20 - 14 - 22, 76 + 14, 22, T.fg, 2.4); return b; }],
];
