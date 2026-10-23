"""A fake Kubernetes API server for the cluster-version fixtures.

  GET /version          -> 200, like a cluster that allows anonymous /version
  GET /locked/version   -> 401 unless "Authorization: Bearer test-token",
                           like a cluster that disables anonymous access
"""
import json
from http.server import BaseHTTPRequestHandler, HTTPServer

VERSION = {
    "major": "1",
    "minor": "30",
    "gitVersion": "v1.30.4-eks-a737599",  # vendor suffix, as managed clusters report it
    "platform": "linux/amd64",
}


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/version":
            return self.reply(200, VERSION)
        if self.path == "/locked/version":
            if self.headers.get("Authorization") == "Bearer test-token":
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


HTTPServer(("0.0.0.0", 8080), Handler).serve_forever()
