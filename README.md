# battleword-cloud-engine
A version of the engine designed to run in the cloud. This is what the UI talks to. Matches are saved into firestore objects.

## Deploying

By making a commit to `api/dev` or `api/prod` this project wil automatically deploy.

## Identity Federation

To allow a github project to use gcloud resources:

Setup pool:
```bash
gcloud iam workload-identity-pools create "github-pool" \
  --project="battleword" \
  --location="global" \
  --display-name="github-pool"
```

Setup workload:
```bash
gcloud iam workload-identity-pools providers create-oidc "github-provider" \
  --project="battleword" \
  --location="global" \
  --workload-identity-pool="github-pool" \
  --display-name="github-provider" \
  --attribute-mapping="google.subject=assertion.sub,attribute.actor=assertion.actor,attribute.aud=assertion.aud,attribute.repository=assertion.repository" \
  --issuer-uri="https://token.actions.githubusercontent.com"
```

Allow the identity provider to impersonate the service account:

```bash
gcloud iam service-accounts add-iam-policy-binding "github@battleword.iam.gserviceaccount.com" \
  --project="battleword" \
  --role="roles/iam.workloadIdentityUser" \
  --member="principalSet://iam.googleapis.com/projects/339690027814/locations/global/workloadIdentityPools/github-pool/attribute.repository/brensch/battleword-cloud-engine"
```
This is kind of magic and tbh I don't understand it well yet. Once it's set up Github is able to use the GCP resources you specify on the service account without any key in our environment.