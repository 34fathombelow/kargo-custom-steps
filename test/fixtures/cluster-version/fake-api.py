"""A fake Kubernetes API server for the cluster-version fixtures.

HTTP, port 8080:
  GET /version          -> 200, like a cluster that allows anonymous /version
  GET /locked/version   -> 401 unless "Authorization: Bearer test-token",
                           like a cluster that disables anonymous access
HTTPS, port 8443 (when /tls/tls.crt is mounted), like k3s:
  GET /version          -> 401 unless "Authorization: Bearer test-token"
"""
import json
import os
import ssl
import threading
from http.server import BaseHTTPRequestHandler, HTTPServer

VERSION = {
    "major": "1",
    "minor": "30",
    "gitVersion": "v1.30.4-eks-a737599",  # vendor suffix, as managed clusters report it
    "platform": "linux/amd64",
}


class Handler(BaseHTTPRequestHandler):
    require_token = False

    def do_GET(self):
        authorized = self.headers.get("Authorization") == "Bearer test-token"
        if self.path == "/version":
            if self.require_token and not authorized:
                return self.reply(401, {"kind": "Status", "message": "Unauthorized"})
            return self.reply(200, VERSION)
        if self.path == "/locked/version":
            if authorized:
                return self.reply(200, VERSION)
            return self.reply(401, {"kind": "Status", "message": "Unauthorized"})
        return self.reply(404, {"kind": "Status", "message": "not found"})

    def reply(self, code, body):
        data = json.dumps(body).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def log_message(self, *args):
        pass


class TokenRequired(Handler):
    require_token = True


if os.path.exists("/tls/tls.crt"):
    tls = HTTPServer(("0.0.0.0", 8443), TokenRequired)
    context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    context.load_cert_chain("/tls/tls.crt", "/tls/tls.key")
    tls.socket = context.wrap_socket(tls.socket, server_side=True)
    threading.Thread(target=tls.serve_forever, daemon=True).start()

HTTPServer(("0.0.0.0", 8080), Handler).serve_forever()
