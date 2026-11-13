#!/usr/bin/env python3
"""Controlled Serper boundary for local integration tests. Never deploy it. Counts calls at /calls."""
import json
import os
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

CALLS = []

class Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        body = json.loads(self.rfile.read(int(self.headers['Content-Length'])))
        CALLS.append(body.get('q'))
        response = json.dumps({'places': [{'title': 'SnowWorld Landgraaf', 'address': 'Witte Wereld 1', 'category': 'Indoor snowcenter', 'website': 'https://www.snowworld.com/landgraaf', 'rating': 4.2}]}).encode()
        self.send_response(200); self.send_header('Content-Type', 'application/json'); self.send_header('Content-Length', str(len(response))); self.end_headers(); self.wfile.write(response)
    def do_GET(self):
        response = json.dumps(CALLS).encode()
        self.send_response(200); self.send_header('Content-Type', 'application/json'); self.send_header('Content-Length', str(len(response))); self.end_headers(); self.wfile.write(response)
    def log_message(self, *args): pass

if __name__ == '__main__':
    ThreadingHTTPServer(('127.0.0.1', int(os.environ.get('SERPER_FIXTURE_PORT', '8182'))), Handler).serve_forever()
