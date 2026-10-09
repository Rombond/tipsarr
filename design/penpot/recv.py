import http.server, sys, os, re
OUT = os.path.expanduser('~/penpot-work/svg')
class H(http.server.BaseHTTPRequestHandler):
    def do_OPTIONS(self):
        self.send_response(204)
        for k, v in (('Access-Control-Allow-Origin','*'),('Access-Control-Allow-Methods','POST,OPTIONS'),('Access-Control-Allow-Headers','*')):
            self.send_header(k, v)
        self.end_headers()
    def do_POST(self):
        name = re.sub(r'[^A-Za-z0-9_.-]', '_', self.path.strip('/')) or 'out'
        n = int(self.headers.get('Content-Length', 0))
        data = self.rfile.read(n)
        open(os.path.join(OUT, name + '.svg'), 'wb').write(data)
        self.send_response(200); self.send_header('Access-Control-Allow-Origin','*'); self.end_headers(); self.wfile.write(b'ok')
    def do_GET(self):
        import os
        name = self.path.strip('/').split('?')[0]
        if name.startswith('boot/'):
            f = os.path.join(os.path.expanduser('~/penpot-work'), name)
            if os.path.isfile(f):
                data = open(f,'rb').read()
                self.send_response(200); self.send_header('Access-Control-Allow-Origin','*'); self.send_header('Content-Type','text/plain; charset=utf-8'); self.end_headers(); self.wfile.write(data); return
        self.send_response(404); self.send_header('Access-Control-Allow-Origin','*'); self.end_headers()
    def log_message(self, *a): pass
http.server.HTTPServer(('127.0.0.1', 8765), H).serve_forever()
