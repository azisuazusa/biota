# Deployment & CI/CD — Dokploy + GitHub Actions — TODO

Reference: `docs/2026-11-23-deployment-dokploy-ci/prd.md`

Publishing a stable `vX.Y.Z` GitHub Release builds `ghcr.io/azisuazusa/biota:<tag>` and deploys it to the Dokploy app at `https://biota.azis.works`. PRs to `main` run build, vet, lint and test.

> Secret handling rule for every task: never `echo`/print `$DOKPLOY_PERSONAL_API_KEY`, and never print a full Dokploy API response that may contain env or secrets. Filter with `jq` to the fields you need.
> Dokploy API base: `https://dokploy.azis.works/api`, header `x-api-key: $DOKPLOY_PERSONAL_API_KEY`, `Content-Type: application/json`.

---

## Phase 1 — Dokploy project & application ✅ DONE

- [x] `GET /project.all`: confirm no project named `biota` exists (only `aptekindo-dashboard`, `doli` expected)
- [x] `POST /project.create` `{"name":"biota","description":"BIOTA marketplace sales insight"}`: record `projectId` → `JnMb-Kxa_UfyvErsK_RgZ`
- [x] `GET /project.all` (filter `biota`): if an environment `production` exists, record its `environmentId`; else `POST /environment.create` `{"name":"production","projectId":"<projectId>"}` — note: `project.create` auto-created `production` → `XwbwH4JYWtMdWu86I11_u`, so environment.create was not needed
- [x] `POST /application.create` `{"name":"biota","appName":"biota","environmentId":"<environmentId>"}`: record `applicationId` (and the `appName` actually assigned) → `QQtqO5CgEMG9tb5z50rxn`; Dokploy assigned appName `biota-bqkl07`
- [x] `POST /application.saveDockerProvider` `{"applicationId":"<id>","dockerImage":"ghcr.io/azisuazusa/biota:v1.0.0","username":"","password":"","registryUrl":""}`. Do **not** call deploy
- [x] `POST /application.saveEnvironment` `{"applicationId":"<id>","env":"PORT=8080\nDATA_DIR=/data\nMAX_UPLOAD_MB=10","buildArgs":"","buildSecrets":"","createEnvFile":false}`
- [x] `POST /mounts.create` `{"type":"volume","volumeName":"biota-data","mountPath":"/data","serviceType":"application","serviceId":"<id>"}`
- [x] `POST /domain.create` `{"host":"biota.azis.works","path":"/","port":8080,"https":true,"certificateType":"letsencrypt","applicationId":"<id>"}`
- [x] `GET /application.one?applicationId=<id>`, using `jq` to print only `sourceType`, `dockerImage`, env keys, mounts `(type, volumeName, mountPath)`, domains `(host, port, https, certificateType)`. Expect docker / `ghcr.io/azisuazusa/biota:v1.0.0` / `PORT, DATA_DIR, MAX_UPLOAD_MB` / `volume biota-data /data` / `biota.azis.works 8080 true letsencrypt`

---

## Phase 2 — GitHub repository & secrets ✅ DONE

- [x] `gh repo view azisuazusa/biota`: confirm it doesn't exist
- [x] `gh repo create azisuazusa/biota --public --description "BIOTA marketplace sales insight dashboard (Go, HTMX, Tailwind, Alpine)"` (no `--source`, no push yet)
- [x] `gh secret set DOKPLOY_URL -R azisuazusa/biota --body "https://dokploy.azis.works"`
- [x] `printf '%s' "$DOKPLOY_PERSONAL_API_KEY" | gh secret set DOKPLOY_API_KEY -R azisuazusa/biota` (stdin; value never printed)
- [x] `gh secret set DOKPLOY_APPLICATION_ID -R azisuazusa/biota --body "<applicationId from Phase 1>"`
- [x] `gh secret list -R azisuazusa/biota`: shows exactly the 3 names — verified; repo https://github.com/azisuazusa/biota is PUBLIC

---

## Phase 3 — Workflows & image fix ✅ DONE

- [x] Precondition: every phase in `docs/2026-11-23-marketplace-sales-insight/task-progress.md` up to QA is done, and `make test` passes. Read `Makefile`, `Dockerfile`, `go.mod` and `.gitignore` first
- [x] `Dockerfile`: make sure the runtime image has `/data` owned by `65532:65532` (e.g. in the build stage `RUN mkdir -p /out/data` and in the runtime stage `COPY --from=build --chown=65532:65532 /out/data /data`), keeping `VOLUME /data` and `USER nonroot`. Note on this task if it's already correct — already correct (subagent added `COPY --from=build --chown=nonroot:nonroot /out/data /data`), no change
- [x] `Dockerfile`: confirm `EXPOSE 8080` and that the binary listens on `$PORT`
- [x] `golangci/golangci-lint-action`: resolve the latest release tag + commit SHA via `gh api repos/golangci/golangci-lint-action/releases/latest --jq .tag_name` and `gh api repos/golangci/golangci-lint-action/commits/<tag> --jq .sha`; use the golangci-lint version the app's `Makefile`/`.golangci.yml` expects (or the latest v2 if unspecified) — resolved v9.3.0 @ `ba0d7d2ec06a0ea1cb5fa41b2e4a3ab91d21278a`, golangci-lint `v2.13.1` (same as local `make run-lint`)
- [x] `.github/workflows/pr.yml`:
  - [x] `name: pr`; `on: pull_request: branches: [main]`; `permissions: contents: read`; `concurrency: { group: pr-${{ github.event.pull_request.number }}, cancel-in-progress: true }`
  - [x] job `checks` (`ubuntu-latest`): `actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1`; `actions/setup-go@924ae3a1cded613372ab5595356fb5720e22ba16 # v6.5.0` with `go-version-file: go.mod`, `cache: true`
  - [x] step "Install templ": `go install github.com/a-h/templ/cmd/templ@$(go list -m -f '{{.Version}}' github.com/a-h/templ)`
  - [x] step "Install Tailwind": same version and URL as the app's `Makefile` `tailwind-install` / Dockerfile (linux-x64), into `bin/tailwindcss`, `chmod +x` — implemented as `make tailwind-install` (Makefile auto-selects linux-x64 on the runner)
  - [x] step "Generate": `templ generate` + Tailwind build to `static/css/app.css --minify` (same command as `make generate`)
  - [x] steps: `go build ./...`, `go vet ./...`, golangci-lint action (pinned SHA, `version:` as resolved), `go test -count=1 ./...`
- [x] `.github/workflows/release.yml`:
  - [x] `name: release`; `on: release: types: [published]`; `permissions: contents: read`; `concurrency: { group: release-deploy, cancel-in-progress: false }`; `env: IMAGE: ghcr.io/azisuazusa/biota`
  - [x] job `gate`: step with `TAG: ${{ github.event.release.tag_name }}`, `PRE: ${{ github.event.release.prerelease }}`; `if [[ "$TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ && "$PRE" == "false" ]]` → `echo deploy=true >> "$GITHUB_OUTPUT"`, else `deploy=false` + `echo "::notice::Skipping: $TAG is not a stable vX.Y.Z release"`. Job output `deploy`
  - [x] job `test`: `needs: gate`, `if: needs.gate.outputs.deploy == 'true'`, checkout `ref: ${{ github.event.release.tag_name }}`, identical steps to `pr.yml` (minus `go build`, which is optional; keep it for parity)
  - [x] job `build`: `needs: [gate, test]`, same `if`, `permissions: { contents: read, packages: write }`; checkout (tag), `docker/setup-buildx-action@f87e5991a6d7451dcb8d9637bfbc97413f497069 # v4.4.1`, `docker/login-action@8aa5831fb8d48d8f76594c7ddde61ae58019f550 # v4.8.0` (`registry: ghcr.io`, `username: ${{ github.actor }}`, `password: ${{ secrets.GITHUB_TOKEN }}`), `docker/metadata-action@dc802804100637a589fabce1cb79ff13a1411302 # v6.2.0` (`images: ${{ env.IMAGE }}`, `tags: type=raw,value=${{ github.event.release.tag_name }}`), `docker/build-push-action@c3c9e263c25d99ce0380d002d59b67737d91b0dc # v7.4.0` (`context: .`, `file: ./Dockerfile`, `platforms: linux/amd64`, `push: true`, tags/labels from meta, `cache-from: type=gha`, `cache-to: type=gha,mode=max`)
  - [x] job `deploy`: `needs: [gate, build]`, same `if`; env `DOKPLOY_URL`, `DOKPLOY_API_KEY`, `DOKPLOY_APPLICATION_ID` from secrets, `TAG`, `RELEASE_URL: ${{ github.event.release.html_url }}`; fail with `::error::` if any secret is empty
    - [x] step "Point Dokploy at image": `body=$(jq -nc --arg id "$DOKPLOY_APPLICATION_ID" --arg img "$IMAGE:$TAG" '{applicationId:$id, dockerImage:$img, username:"", password:"", registryUrl:""}')`; `curl --fail-with-body -sS -X POST "${DOKPLOY_URL%/}/api/application.saveDockerProvider" -H 'accept: application/json' -H 'Content-Type: application/json' -H "x-api-key: ${DOKPLOY_API_KEY}" -d "$body" -o /dev/null -w 'saveDockerProvider HTTP %{http_code}\n'`
    - [x] step "Trigger deploy": `body=$(jq -nc --arg id "$DOKPLOY_APPLICATION_ID" --arg t "Release $TAG" --arg d "$RELEASE_URL" '{applicationId:$id, title:$t, description:$d}')`; `curl … /api/application.deploy … -o /dev/null -w 'deploy HTTP %{http_code}\n'`
- [x] Validate YAML locally: `actionlint .github/workflows/*.yml` (install via `go install github.com/rhysd/actionlint/cmd/actionlint@latest`), or note if unavailable — actionlint passes (fixed SC2046 by quoting the templ install arg)
- [x] `README.md`: add a "Deployment" section covering how to release (`gh release create vX.Y.Z --generate-notes`), what gets deployed, the tag rules, rollback (re-run an older release's workflow), and the 3 secrets (names only)

---

## Phase 4 — DNS (owner, manual) ✅ DONE

- [x] Owner adds an A record `biota.azis.works → 43.163.102.93`
- [x] `dig +short biota.azis.works` returns `43.163.102.93` — verified 2026-11-23 via local resolver, 1.1.1.1 and 8.8.8.8

---

## Phase 5 — Go-live

- [ ] `.gitignore`: confirm it excludes `bin/`, `.data/`, `*_templ.go`, `static/css/app.css`, `.env*`, `.DS_Store`
- [ ] `git init -b main` in `/Users/azisuazusa/Projects/biota`
- [ ] `git status`: review the staged file list; make sure no secrets, `.data/`, binaries or `.claude/settings.local.json` are included
- [ ] Initial commit: `git add -A && git commit` with message `Initial commit: BIOTA marketplace sales insight app + CI/CD` ending with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`
- [ ] `git remote add origin git@github.com:azisuazusa/biota.git && git push -u origin main`
- [ ] Optional sanity check: open a throwaway PR (e.g. a README typo branch) to see `pr.yml` go green, then close it. Skip if the owner prefers, with a note
- [ ] `gh release create v1.0.0 --title "v1.0.0" --generate-notes -R azisuazusa/biota`
- [ ] `gh run watch` on the `release` run: gate → test → build → deploy
- [ ] GHCR visibility: anonymously check `TOKEN=$(curl -s "https://ghcr.io/token?scope=repository:azisuazusa/biota:pull" | jq -r .token)`; `curl -s -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $TOKEN" -H 'Accept: application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.v2+json' https://ghcr.io/v2/azisuazusa/biota/manifests/v1.0.0` → expect `200`
  - [ ] If not 200: ask the owner to set the package public at `https://github.com/users/azisuazusa/packages/container/biota/settings`, then re-check and `gh run rerun <run-id> --failed`

---

## Phase 6 — QA

- [ ] `pr.yml` ran green on a PR (or noted as skipped in Phase 5)
- [ ] `release.yml` for `v1.0.0`: all 4 jobs green
- [ ] `GET /application.one`: `dockerImage` = `ghcr.io/azisuazusa/biota:v1.0.0`, and the latest deployment status is `done`
- [ ] `curl -sS -o /dev/null -w '%{http_code} %{ssl_verify_result}\n' https://biota.azis.works/healthz` → `200 0` (valid cert)
- [ ] Browser: `https://biota.azis.works/` shows the empty state; upload `test_data/sample_orders_oct2026.csv` → confirm → dashboard shows Pendapatan Bersih Rp360.363.000
- [ ] Redeploy in Dokploy (`POST /application.deploy`) → dashboard still shows Rp360.363.000 (the volume persists `dataset.json` + `config.json`)
- [ ] Prerelease check: `gh release create v1.0.1-rc.1 --prerelease --notes "test"` → `gate` skips build/deploy; then delete that release and tag (`gh release delete v1.0.1-rc.1 --cleanup-tag -y`)
- [ ] `gh run view --log` of the release run contains no secret values (the API key is masked as `***`)
