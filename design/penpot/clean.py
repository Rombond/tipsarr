import sys, xml.etree.ElementTree as ET
ET.register_namespace('', 'http://www.w3.org/2000/svg'); ET.register_namespace('xlink', 'http://www.w3.org/1999/xlink')
src, dst = sys.argv[1], sys.argv[2]
tree = ET.parse(src); root = tree.getroot()
ns = '{http://www.w3.org/2000/svg}'
gs = [c for c in list(root) if c.tag == ns + 'g']
for g in gs[1:]:
    root.remove(g)
# crop to the root board: first clipPath rect
cp = root.find('.//' + ns + 'clipPath')
if cp is not None:
    r = cp.find(ns + 'rect')
    if r is not None:
        x, y, w, h = (float(r.get(k)) for k in ('x', 'y', 'width', 'height'))
        root.set('viewBox', f'{x} {y} {w} {h}'); root.set('width', str(w)); root.set('height', str(h))
tree.write(dst, xml_declaration=False)
