#!/usr/bin/env python
import os

from constructs import Construct
from cdk8s import App, Chart

from imports import k8s

# The Kargo cdk8s-synth step passes per-Stage settings as environment variables.
environment = os.environ.get("ENVIRONMENT", "dev")
image = os.environ.get("IMAGE", "ghcr.io/akuity/guestbook:v0.0.2")


class Guestbook(Chart):
    def __init__(self, scope: Construct, id: str):
        super().__init__(scope, id)

        labels = {"app": "guestbook-ui", "environment": environment}

        k8s.KubeDeployment(
            self,
            "deployment",
            metadata=k8s.ObjectMeta(name="guestbook-ui", labels=labels),
            spec=k8s.DeploymentSpec(
                replicas=3 if environment == "prod" else 1,
                selector=k8s.LabelSelector(match_labels={"app": "guestbook-ui"}),
                template=k8s.PodTemplateSpec(
                    metadata=k8s.ObjectMeta(labels=labels),
                    spec=k8s.PodSpec(
                        containers=[
                            k8s.Container(
                                name="guestbook-ui",
                                image=image,
                                ports=[k8s.ContainerPort(container_port=80)],
                            )
                        ]
                    ),
                ),
            ),
        )

        k8s.KubeService(
            self,
            "service",
            metadata=k8s.ObjectMeta(name="guestbook-ui", labels=labels),
            spec=k8s.ServiceSpec(
                selector={"app": "guestbook-ui"},
                ports=[k8s.ServicePort(port=80, target_port=k8s.IntOrString.from_number(80))],
            ),
        )


app = App()
Guestbook(app, "guestbook")
app.synth()
