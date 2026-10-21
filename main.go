package main

import (
	"os"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/cmd"
)

// GroupName is the API group the webhook registers under. It must match the
// APIService and the customer's Issuer solver config (e.g. acme.enum.co); the
// chart sets it via the GROUP_NAME env.
var GroupName = os.Getenv("GROUP_NAME")

func main() {
	if GroupName == "" {
		panic("GROUP_NAME must be specified")
	}

	cmd.RunWebhookServer(GroupName, &enumSolver{})
}
