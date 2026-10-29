package main

import (
	"os"

	"example.com/cdk8s-synth-demo/imports/k8s"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
)

// The Kargo cdk8s-synth steps pass per-Stage settings as environment variables.
func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func NewGuestbook(scope constructs.Construct, id string) cdk8s.Chart {
	chart := cdk8s.NewChart(scope, jsii.String(id), nil)

	env := getenv("ENVIRONMENT", "dev")
	image := getenv("IMAGE", "ghcr.io/akuity/guestbook:v0.0.2")
	labels := &map[string]*string{"app": jsii.String("guestbook-ui"), "environment": jsii.String(env)}

	k8s.NewKubeDeployment(chart, jsii.String("deployment"), &k8s.KubeDeploymentProps{
		Metadata: &k8s.ObjectMeta{Name: jsii.String("guestbook-ui"), Labels: labels},
		Spec: &k8s.DeploymentSpec{
			Replicas: "three", // intentional type error for the cdk8s-synth tests
			Selector: &k8s.LabelSelector{MatchLabels: &map[string]*string{"app": jsii.String("guestbook-ui")}},
			Template: &k8s.PodTemplateSpec{
				Metadata: &k8s.ObjectMeta{Labels: labels},
				Spec: &k8s.PodSpec{
					Containers: &[]*k8s.Container{{
						Name:  jsii.String("guestbook-ui"),
						Image: jsii.String(image),
						Ports: &[]*k8s.ContainerPort{{ContainerPort: jsii.Number(80)}},
					}},
				},
			},
		},
	})

	k8s.NewKubeService(chart, jsii.String("service"), &k8s.KubeServiceProps{
		Metadata: &k8s.ObjectMeta{Name: jsii.String("guestbook-ui"), Labels: labels},
		Spec: &k8s.ServiceSpec{
			Selector: &map[string]*string{"app": jsii.String("guestbook-ui")},
			Ports: &[]*k8s.ServicePort{{
				Port:       jsii.Number(80),
				TargetPort: k8s.IntOrString_FromNumber(jsii.Number(80)),
			}},
		},
	})

	return chart
}

func main() {
	app := cdk8s.NewApp(nil)
	NewGuestbook(app, "guestbook")
	app.Synth()
}
