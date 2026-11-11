#!/usr/bin/env python3
"""Loopback-only native browser-return fixture. It never authenticates a user."""
from http.server import BaseHTTPRequestHandler,ThreadingHTTPServer
class Handler(BaseHTTPRequestHandler):
 def do_GET(self):
  if self.path=='/finish':
   self.send_response(302);self.send_header('Location','io.liveleds.minder:/oauth/callback?state=emulator-probe&code=not-an-auth-code');self.end_headers();return
  content=b'''<!doctype html><meta name="viewport" content="width=device-width,initial-scale=1"><title>Minder callback check</title><body style="padding:30px;font:20px sans-serif"><p>Emulator browser-return check. No account or session.</p><a style="display:block;padding:24px;background:#ddd" href="/finish">Return to Minder</a></body>'''
  self.send_response(200);self.send_header('Content-Type','text/html');self.send_header('Content-Length',str(len(content)));self.end_headers();self.wfile.write(content)
 def log_message(self,*args):pass
if __name__=='__main__':ThreadingHTTPServer(('127.0.0.1',8184),Handler).serve_forever()
