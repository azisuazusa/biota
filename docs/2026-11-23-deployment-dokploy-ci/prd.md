# PRD: Deployment & CI/CD — Dokploy + GitHub Actions

**Date:** 2026-11-23
**Status:** Draft

---

## 1. Problem Statement

The BIOTA Marketplace Sales Insight app (`docs/2026-11-23-marketplace-sales-insight/prd.md`) is being built in `/Users/azisuazusa/Projects/biota` but cannot be deployed yet:
- The folder is not a git repository, and `azisuazusa/biota` does not exist on GitHub.
- No CI exists, so pull requests aren't checked and releases aren't built.
- The Dokploy server at `https://dokploy.azis.works` has only two projects, `aptekindo-dashboard` and `doli`. There's no project or application for biota.
- `biota.azis.works` has no DNS record, and `*.azis.works` has no wildcard. `dokploy.azis.works` and `aptekindo.azis.works` resolve to `43.163.102.93`.

This feature sets up the full path from a published GitHub Release to a running app at `https://biota.azis.works`.

---

## 2. Why This Matters

Without it, the app runs only on a laptop, which defeats its purpose as a dashboard the owner opens after each marketplace export. Manual deploys (building images locally, clicking through Dokploy) are error-prone and leave no record of what version is live. A tag-pinned, release-driven pipeline makes every deploy traceable to a version (`v1.2.3`) and lets you roll back.

---

## 3. Who Is This For

**Primary:** The repo owner (`azisuazusa`). They publish GitHub Releases to deploy, open PRs to get checks, and use the app at `https://biota.azis.works`.
**Secondary:** Anyone browsing the public repo. They can read the code, and the workflows run on their PRs (the PR workflow is read-only and needs no secrets).

---

## 4. What Success Looks Like

| Metric | Baseline | Target | Timeframe |
|--------|----------|--------|-----------|
| Public repo `github.com/azisuazusa/biota` with app code on `main` | Doesn't exist | Exists, public, default branch `main` | At launch |
| PR checks | None | `pr.yml` runs build, vet, golangci-lint and test on every PR to `main` | At launch |
| Release `v1.0.0` published → live | N/A | `release.yml` green; Dokploy app runs `ghcr.io/azisuazusa/biota:v1.0.0` | At launch |
| `https://biota.azis.works/healthz` | No DNS | HTTP 200, valid Let's Encrypt certificate | At launch (after DNS) |
| Image pull without credentials | N/A | `ghcr.io/azisuazusa/biota:v1.0.0` pulls anonymously | At launch |
| Data persists across redeploys | N/A | After uploading the sample CSV and redeploying, the dashboard still shows Rp360.363.000 | At launch |
| Non-semver or prerelease releases | N/A | Workflow runs, but build and deploy are **skipped** | At launch |
| Secrets exposure | N/A | No secret value appears in any log, doc, or commit | Always |

---

## 5. Scope

### In Scope

**GitHub repository**
- Created with `gh repo create azisuazusa/biota --public --description "BIOTA marketplace sales insight dashboard (Go, HTMX, Tailwind, Alpine)"`. Default branch `main`. `gh` is authenticated as `azisuazusa` and uses the SSH git protocol.
- Remote `origin` = `git@github.com:azisuazusa/biota.git`.
- Repo secrets, used only by `release.yml`:
  | Secret | Value | How it's set |
  |---|---|---|
  | `DOKPLOY_URL` | `https://dokploy.azis.works` | `gh secret set` |
  | `DOKPLOY_API_KEY` | the value of local env var `DOKPLOY_PERSONAL_API_KEY`. Decided: reuse the same key | piped from the env var into `gh secret set` via stdin, never echoed |
  | `DOKPLOY_APPLICATION_ID` | `applicationId` returned by `application.create` | `gh secret set` |
- Secrets aren't exposed to `pull_request` runs from forks (GitHub default), and `pr.yml` doesn't reference them.

**Container registry**
- GHCR image `ghcr.io/azisuazusa/biota`, tagged with the **exact release tag** (e.g. `v1.0.0`). No `:latest` tag. Decided: Dokploy runs the exact version, so what's live is always identifiable.
- The package must be **public** so Dokploy pulls it without credentials. Dokploy's `saveDockerProvider` gets empty `username`, `password` and `registryUrl`.
- Constraint: a newly created GHCR container package can default to **private**, and GitHub has no REST API to change a container package's visibility. After the first push, check whether anonymous pull works. If it doesn't, the owner sets visibility to Public once at `https://github.com/users/azisuazusa/packages/container/biota/settings`, then re-runs the failed deploy job. The image carries `org.opencontainers.image.source=https://github.com/azisuazusa/biota` (added by `docker/metadata-action`), which links the package to the repo.

**PR workflow — `.github/workflows/pr.yml`**
- Trigger: `pull_request` targeting `main`.
- Permissions: `contents: read`.
- One job `checks` on `ubuntu-latest`: checkout → setup-go (`go-version-file: go.mod`, cache) → install templ (version pinned to the one in `go.mod`) → `templ generate` → install Tailwind standalone CLI (same version/method as the app's `Makefile`/`Dockerfile`) → build `static/css/app.css` → `go build ./...` → `go vet ./...` → `golangci-lint` (via `golangci/golangci-lint-action`, pinned SHA) → `go test -count=1 ./...`.
- Decided: **go build only**, with no Docker build in PRs.
- `concurrency: group: pr-${{ github.event.pull_request.number }}`, `cancel-in-progress: true`.

**Release workflow — `.github/workflows/release.yml`**
- Trigger: `release: types: [published]`.
- Gate job `gate`: outputs `deploy=true` only if `github.event.release.tag_name` matches `^v[0-9]+\.[0-9]+\.[0-9]+$` **and** `github.event.release.prerelease == false`. Otherwise it writes a notice ("Skipping: tag … is not a stable vX.Y.Z release") and the downstream jobs are skipped.
- `concurrency: group: release-deploy`, `cancel-in-progress: false`, so deploys never overlap.
- Jobs (each `needs` the previous and runs only when `needs.gate.outputs.deploy == 'true'`):
  | Job | Steps | Permissions |
  |---|---|---|
  | `test` | Same steps as `pr.yml`: generate, go vet, golangci-lint, go test | `contents: read` |
  | `build` | checkout (ref = tag) → setup-buildx → login to ghcr.io with `GITHUB_TOKEN` → metadata-action (`images: ghcr.io/azisuazusa/biota`, `tags: type=raw,value=${{ github.event.release.tag_name }}`) → build-push-action (`platforms: linux/amd64`, `push: true`, GHA cache) | `contents: read`, `packages: write` |
  | `deploy` | ① `POST /api/application.saveDockerProvider` with `{applicationId, dockerImage: "ghcr.io/azisuazusa/biota:<tag>", username: "", password: "", registryUrl: ""}` ② `POST /api/application.deploy` with `{applicationId, title: "Release <tag>", description: "<release html_url>"}`. Both use `curl --fail-with-body -sS`, header `x-api-key`, and a JSON body built with `jq -nc` | none beyond default |
- **No manual approval**: publishing the Release is the approval. No GitHub Environment protection rules.
- The deploy step fails with `::error::` if any of the 3 secrets is empty, following the aptekindo pattern.
- **Rollback**: re-run the `release.yml` run of an older release ("Re-run all jobs"), which points Dokploy back at that older tag. Alternatively, call `saveDockerProvider` + `deploy` by hand.

**Action versions** (pinned by SHA, the same as the reference `azisuazusa/aptekindo-dashboard/.github/workflows/deploy.yml`)
| Action | SHA | Version |
|---|---|---|
| `actions/checkout` | `3d3c42e5aac5ba805825da76410c181273ba90b1` | v7.0.1 |
| `actions/setup-go` | `924ae3a1cded613372ab5595356fb5720e22ba16` | v6.5.0 |
| `docker/setup-buildx-action` | `f87e5991a6d7451dcb8d9637bfbc97413f497069` | v4.4.1 |
| `docker/login-action` | `8aa5831fb8d48d8f76594c7ddde61ae58019f550` | v4.8.0 |
| `docker/metadata-action` | `dc802804100637a589fabce1cb79ff13a1411302` | v6.2.0 |
| `docker/build-push-action` | `c3c9e263c25d99ce0380d002d59b67737d91b0dc` | v7.4.0 |
| `golangci/golangci-lint-action` | resolve the latest release's commit SHA when implementing | latest stable |

**Dokploy setup** (one-time, through the API at `https://dokploy.azis.works/api/*` with header `x-api-key: $DOKPLOY_PERSONAL_API_KEY`. Endpoints verified against the server's `settings.getOpenApiDocument`.)
| Step | Endpoint | Body |
|---|---|---|
| 1 | `POST project.create` | `{"name":"biota","description":"BIOTA marketplace sales insight"}`. If the response or `project.all` already shows an environment named `production`, use it |
| 2 | `POST environment.create` (only if step 1 created none) | `{"name":"production","projectId":"…"}` |
| 3 | `POST application.create` | `{"name":"biota","appName":"biota","environmentId":"…"}`. Record `applicationId` |
| 4 | `POST application.saveDockerProvider` | `{"applicationId":"…","dockerImage":"ghcr.io/azisuazusa/biota:v1.0.0","username":"","password":"","registryUrl":""}`. The image doesn't exist yet and **no deploy is triggered here**. The release workflow overwrites the image on every release |
| 5 | `POST application.saveEnvironment` | `{"applicationId":"…","env":"PORT=8080\nDATA_DIR=/data\nMAX_UPLOAD_MB=10","buildArgs":"","buildSecrets":"","createEnvFile":false}` |
| 6 | `POST mounts.create` | `{"type":"volume","volumeName":"biota-data","mountPath":"/data","serviceType":"application","serviceId":"<applicationId>"}` |
| 7 | `POST domain.create` | `{"host":"biota.azis.works","path":"/","port":8080,"https":true,"certificateType":"letsencrypt","applicationId":"…"}` |
- Verify with `GET application.one?applicationId=…`: sourceType `docker`, the image, env, 1 volume mount, 1 domain.
- Volume permissions: the app image runs as distroless `nonroot` (UID 65532). Docker copies an image directory's ownership into a **new, empty named volume** on first mount. So the Dockerfile must create `/data` owned by `65532:65532`, or the app can't write `config.json` and `dataset.json`. If the app's Dockerfile doesn't already do this, it's changed as part of this feature.

**DNS**
- Done by the owner, not automated: add an **A record** `biota.azis.works → 43.163.102.93` at the DNS provider.
- Let's Encrypt (HTTP-01 through Traefik in Dokploy) only succeeds once the record resolves. If the certificate was requested before DNS propagated, re-trigger it by redeploying the app or updating the domain.

**Go-live sequence**
1. Wait until the app implementation (in progress, `docs/2026-11-23-marketplace-sales-insight/task-progress.md`) is complete and `make test` passes locally.
2. Add the workflows (and the Dockerfile `/data` ownership fix if needed).
3. `git init -b main`, `.gitignore` check (no `.data/`, `bin/`, generated files, or secrets), initial commit, `git remote add origin`, `git push -u origin main`.
4. Owner adds the DNS record.
5. Publish Release **`v1.0.0`**: `gh release create v1.0.0 --title "v1.0.0" --generate-notes`.
6. Watch `release.yml`. If the deploy can't pull the image, apply the GHCR visibility step and re-run the deploy job.
7. Verify (see §4).

### Out of Scope
- **Docker build in PR checks.** Decided: go build only.
- **Manual approval / GitHub Environment protection.** Decided: none.
- **`:latest` image tag.** Decided: exact version tag only.
- **Deploying prereleases or non-semver tags.** Decided: skipped.
- **Deploy on push to `main`.** Only published Releases deploy.
- **Automating DNS.** The owner adds the A record.
- **A separate, dedicated Dokploy API key.** Decided: reuse `DOKPLOY_PERSONAL_API_KEY`.
- **Post-deploy smoke test inside the workflow.** Verification is manual (see §4).
- **Multi-arch images.** linux/amd64 only, matching aptekindo on the same server.
- **Staging environment, preview deployments, backups of the `biota-data` volume.**
- **Branch protection rules.**

---

## 6. Constraints & Dependencies

- Depends on the app implementation being complete, including `Dockerfile`, `Makefile`, `/healthz`, and `go.mod` with module `github.com/azisuazusa/biota`.
- The local Docker daemon isn't running on the dev Mac, so the image build is validated only in CI (`release.yml`).
- Tooling in CI must match the app's local tooling: the templ version in `go.mod`, and the Tailwind standalone CLI version and download method used by `Makefile`/`Dockerfile`. No npm.
- Secrets are never printed in terminal output, docs, commit messages, or workflow logs. They're piped via stdin (`printf '%s' "$DOKPLOY_PERSONAL_API_KEY" | gh secret set DOKPLOY_API_KEY -R azisuazusa/biota`), and Dokploy API responses that could contain secrets aren't echoed.
- `release` events run the workflow file **as it exists at the tagged commit**, so workflow fixes need a new release, or a re-run after re-tagging.
- The Dokploy API `appName` must be unique on the server. If `biota` is taken, use the value Dokploy returns.
- Commits end with the attribution trailer `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.

---

## 7. Related Docs

| Document | Purpose |
|----------|---------|
| `docs/2026-11-23-marketplace-sales-insight/prd.md` | The app being deployed (env vars, `/healthz`, data volume) |
| `docs/2026-11-23-marketplace-sales-insight/task-progress.md` | App implementation progress, which must finish first |
| `docs/2026-11-23-deployment-dokploy-ci/task-progress.md` | Implementation plan for this PRD |
| `github.com/azisuazusa/aptekindo-dashboard/.github/workflows/deploy.yml` | Reference workflow (pinned actions, Dokploy deploy call) |

---

## 8. Open Questions

None at this time.
