# cert-manager-webhook-enum

A [cert-manager](https://cert-manager.io) DNS-01 webhook solver for DNS zones
hosted on [enum](https://enum.co). It lets you issue certificates, including
wildcards, for domains whose DNS you manage on enum.

The solver creates and removes the `_acme-challenge` TXT record through the enum
public API, so the challenge record is stored in enum's database and shown in
your enum dashboard like any other record.

## Prerequisites

- cert-manager installed in the cluster.
- An enum DNS zone that is already active (its nameservers are delegated to
  enum). A challenge can only validate against a live zone.
- A project-scoped enum API token.

## Install

```bash
helm install cert-manager-webhook-enum \
  oci://registry.enum.cloud/enum/charts/cert-manager-webhook-enum \
  --namespace cert-manager
```

Install into the same namespace as cert-manager so the default RBAC lines up.

## Configure

1. Store your enum API token in a Secret. For a `ClusterIssuer`, put it in the
   namespace where cert-manager resolves issuer credentials (its
   cluster-resource-namespace, `cert-manager` by default):

   ```bash
   kubectl -n cert-manager create secret generic enum-api-token \
     --from-literal=token=<YOUR_ENUM_API_TOKEN>
   ```

2. Reference the solver in an `Issuer` or `ClusterIssuer`:

   ```yaml
   apiVersion: cert-manager.io/v1
   kind: ClusterIssuer
   metadata:
     name: enum-letsencrypt
   spec:
     acme:
       server: https://acme-v02.api.letsencrypt.org/directory
       email: you@example.com
       privateKeySecretRef:
         name: enum-letsencrypt-account-key
       solvers:
         - dns01:
             webhook:
               groupName: acme.enum.co
               solverName: enum
               config:
                 projectId: <YOUR_ENUM_PROJECT_ID>
                 apiTokenSecretRef:
                   name: enum-api-token
                   key: token
   ```

3. Request a certificate:

   ```yaml
   apiVersion: cert-manager.io/v1
   kind: Certificate
   metadata:
     name: example-com
   spec:
     secretName: example-com-tls
     issuerRef:
       name: enum-letsencrypt
       kind: ClusterIssuer
     dnsNames:
       - example.com
       - "*.example.com"
   ```

### Solver config

| Field               | Required | Description                                                        |
| ------------------- | -------- | ------------------------------------------------------------------ |
| `projectId`         | yes      | The enum project that owns the zone.                               |
| `apiTokenSecretRef` | yes      | `{name, key}` of the Secret holding the API token.                 |
| `apiUrl`            | no       | enum public API endpoint. Defaults to `api.enum.co:443`.           |

If your token Secret uses a name other than `enum-api-token`, add it to
`secretNames` in the chart values so the webhook is allowed to read it.
