"""A fake Sentry API for the sentry-release fixtures: the three calls the step makes,
with the responses and errors real Sentry returns. Deterministic across runs, so
fixtures can assert exact outputs.

  POST /api/0/organizations/acme/releases/                  201 new, 208 for SEEDED versions
  PUT  /api/0/organizations/acme/releases/<version>/        refs (commits) and dateReleased
  POST /api/0/organizations/acme/releases/<version>/deploys/

Token "bad" -> 401. Org other than acme -> 404. Project other than guestbook -> 400.
Repository other than acme/guestbook in refs -> 400.
"""
import hashlib
import json
from http.server import BaseHTTPRequestHandler, HTTPServer
from urllib.parse import unquote

ORG, PROJECT, PROJECT_ID, REPO = "acme", "guestbook", 4512342186066000, "acme/guestbook"
SEEDED = {"guestbook@1.0.0"}  # releases that already exist
commits = {}  # version -> commitCount set by the last refs update


def release(version, released=None):
    return {"version": version, "dateReleased": released, "commitCount": commits.get(version, 0),
            "projects": [{"id": PROJECT_ID, "slug": PROJECT, "name": PROJECT}]}


class Handler(BaseHTTPRequestHandler):
    def reply(self, code, body):
        data = json.dumps(body).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def route(self, method):
        if self.headers.get("Authorization") == "Bearer bad":
            return self.reply(401, {"detail": "Invalid token"})
        parts = self.path.split("?")[0].strip("/").split("/")  # api 0 organizations <org> releases [<v>] [deploys]
        if parts[:3] != ["api", "0", "organizations"] or len(parts) < 5 or parts[4] != "releases":
            return self.reply(404, {"detail": "The requested resource does not exist"})
        if parts[3] != ORG:
            return self.reply(404, {"detail": "The requested resource does not exist"})
        body = json.loads(self.rfile.read(int(self.headers.get("Content-Length") or 0)) or b"{}")
        if method == "POST" and len(parts) == 5:
            if body.get("projects") != [PROJECT]:
                return self.reply(400, {"projects": ["Invalid project ids or slugs"]})
            version = body["version"]
            return self.reply(208 if version in SEEDED else 201, release(version))
        version = unquote(parts[5])
        if method == "PUT" and len(parts) == 6:
            for ref in body.get("refs", []):
                if ref.get("repository") != REPO:
                    return self.reply(400, {"refs": [f"Invalid repository names: {ref.get('repository')}"]})
                commits[version] = 3 if ref.get("previousCommit") else 1
            return self.reply(200, release(version, body.get("dateReleased")))
        if method == "POST" and parts[6:] == ["deploys"]:
            key = f"{version}/{body.get('environment')}/{body.get('name')}"
            deploy_id = str(int(hashlib.sha256(key.encode()).hexdigest()[:7], 16))
            return self.reply(201, {"id": deploy_id, "environment": body.get("environment"), "name": body.get("name")})
        return self.reply(404, {"detail": "The requested resource does not exist"})

    def do_POST(self):
        self.route("POST")

    def do_PUT(self):
        self.route("PUT")

    def log_message(self, *args):
        pass


HTTPServer(("0.0.0.0", 8000), Handler).serve_forever()
