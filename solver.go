package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	clientgo "github.com/enumco/client-go"
	apiv1 "github.com/enumco/proto-gen-go/gen/enum/api/v1"
	"github.com/cert-manager/cert-manager/pkg/acme/webhook"
	whapi "github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"golang.org/x/net/publicsuffix"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var _ webhook.Solver = (*enumSolver)(nil)

// challengeTTL is the TTL for the _acme-challenge TXT record. The record is
// transient and removed as soon as the challenge validates, so it is short.
const challengeTTL = 60

// apiTimeout bounds each Present/CleanUp call's interaction with the enum API.
const apiTimeout = 30 * time.Second

// enumSolver is the cert-manager DNS-01 webhook solver. Present/CleanUp create
// and delete the _acme-challenge TXT record through the enum public DNS API, so
// the challenge lands in the enum DB (single write path) rather than being
// written to PowerDNS directly.
type enumSolver struct {
	kube kubernetes.Interface
}

// Name is the solver name customers reference via solverName in their Issuer.
func (s *enumSolver) Name() string {
	return "enum"
}

func (s *enumSolver) Initialize(kubeClientConfig *rest.Config, _ <-chan struct{}) error {
	kube, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		return fmt.Errorf("build kubernetes client: %w", err)
	}
	s.kube = kube
	return nil
}

func (s *enumSolver) Present(ch *whapi.ChallengeRequest) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
	defer cancel()

	c, projectID, zoneID, err := s.prepare(ctx, ch)
	if err != nil {
		return err
	}
	defer c.Close()

	name := recordName(ch.ResolvedFQDN)
	content := strconv.Quote(ch.Key)

	// AddRecordSetValue is atomic and idempotent server-side: it creates the
	// _acme-challenge TXT rrset or adds this value to it under a per-zone lock,
	// so a concurrent wildcard + apex challenge at the same name cannot clobber
	// each other, and a repeated Present is a no-op.
	if _, err := c.DNS.AddRecordSetValue(ctx, &apiv1.AddRecordSetValueRequest{
		ProjectId: projectID,
		ZoneId:    zoneID,
		Name:      name,
		Type:      "TXT",
		Ttl:       challengeTTL,
		Value:     &apiv1.DnsRecordValue{Content: content},
	}); err != nil {
		return fmt.Errorf("add challenge value: %w", err)
	}
	return nil
}

func (s *enumSolver) CleanUp(ch *whapi.ChallengeRequest) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
	defer cancel()

	c, projectID, zoneID, err := s.prepare(ctx, ch)
	if err != nil {
		return err
	}
	defer c.Close()

	name := recordName(ch.ResolvedFQDN)
	content := strconv.Quote(ch.Key)

	// RemoveRecordSetValue is atomic and idempotent server-side: it drops this
	// value from the _acme-challenge TXT rrset under a per-zone lock (deleting
	// the rrset once its last value is gone), leaving a concurrent challenge's
	// value untouched.
	if _, err := c.DNS.RemoveRecordSetValue(ctx, &apiv1.RemoveRecordSetValueRequest{
		ProjectId: projectID,
		ZoneId:    zoneID,
		Name:      name,
		Type:      "TXT",
		Content:   content,
	}); err != nil {
		return fmt.Errorf("remove challenge value: %w", err)
	}

	return nil
}

// prepare loads the per-Issuer config, reads the API token from its Secret,
// opens the enum API client, and resolves the challenge zone to its ID. The
// caller owns closing the returned client.
func (s *enumSolver) prepare(ctx context.Context, ch *whapi.ChallengeRequest) (*clientgo.Client, string, string, error) {
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return nil, "", "", err
	}

	token, err := s.token(ctx, ch.ResourceNamespace, cfg.APITokenSecretRef)
	if err != nil {
		return nil, "", "", err
	}

	addr := cfg.APIURL
	if addr == "" {
		addr = defaultAPIURL
	}

	c, err := clientgo.New(clientgo.WithAddr(addr), clientgo.WithToken(token))
	if err != nil {
		return nil, "", "", fmt.Errorf("create enum api client: %w", err)
	}

	zoneName, err := zoneNameForFQDN(ch.ResolvedFQDN)
	if err != nil {
		c.Close()
		return nil, "", "", fmt.Errorf("derive zone for %q: %w", ch.ResolvedFQDN, err)
	}

	resp, err := c.DNS.GetZoneByName(ctx, &apiv1.GetZoneByNameRequest{ProjectId: cfg.ProjectID, Name: zoneName})
	if err != nil {
		c.Close()
		return nil, "", "", fmt.Errorf("lookup zone %q: %w", zoneName, err)
	}

	return c, cfg.ProjectID, resp.Zone.Id, nil
}

// zoneNameForFQDN derives the registrable root domain (eTLD+1) that owns a
// challenge FQDN, matching the derivation the enum API and enumctl use.
func zoneNameForFQDN(fqdn string) (string, error) {
	return publicsuffix.EffectiveTLDPlusOne(strings.TrimSuffix(strings.ToLower(fqdn), "."))
}

// recordName canonicalizes a challenge FQDN to the form enum stores (lowercase,
// trailing dot).
func recordName(fqdn string) string {
	n := strings.ToLower(fqdn)
	if !strings.HasSuffix(n, ".") {
		n += "."
	}

	return n
}
