"""A fake Argo CD application-controller metrics endpoint for the cluster-version
fixtures. Serves Prometheus text like argocd-metrics:8082/metrics does.

  /metrics             one cluster (what each Akuity Argo CD agent reports)
  /multi/metrics       several clusters (a controller managing more than one)
  /minor-only/metrics  k8s_version without a patch number, as some Argo CD versions report it
  /no-cluster/metrics  Argo CD metrics, but no argocd_cluster_info
"""
from http.server import BaseHTTPRequestHandler, HTTPServer

PREAMBLE = (
    "# HELP argocd_app_info Information about application.\n"
    "# TYPE argocd_app_info gauge\n"
    'argocd_app_info{name="guestbook",namespace="argocd",sync_status="Synced"} 1\n'
    "# HELP argocd_cluster_info Information about cluster.\n"
    "# TYPE argocd_cluster_info gauge\n"
)
INFO = 'argocd_cluster_info{{k8s_version="{v}",name="{n}",server="{s}"}} 1\n'

PAGES = {
    "/metrics": PREAMBLE + INFO.format(v="v1.35.5", n="custom-steps-agent", s="http://cluster-custom-steps-agent:8001"),
    "/multi/metrics": PREAMBLE
    + INFO.format(v="v1.31.4", n="prod", s="http://cluster-prod:8001")
    + INFO.format(v="v1.34.2", n="test", s="http://cluster-test:8001")
    # label order varies in the wild; the step must not depend on it
    + 'argocd_cluster_info{name="dev",server="http://cluster-dev:8001",k8s_version="v1.34.2+k3s1"} 1\n',
    "/minor-only/metrics": PREAMBLE + INFO.format(v="1.34", n="legacy", s="https://legacy.example.com"),
    "/no-cluster/metrics": PREAMBLE.split("# HELP argocd_cluster_info")[0],
}


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        body = PAGES.get(self.path)
        code = 200 if body is not None else 404
        data = (body or "not found\n").encode()
        self.send_response(code)
        self.send_header("Content-Type", "text/plain; version=0.0.4")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def log_message(self, *args):
        pass


HTTPServer(("0.0.0.0", 8082), Handler).serve_forever()
