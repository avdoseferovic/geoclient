# Deployment

This repo is designed so runtime assets are never committed to GitHub.

The deployment model uses two independent version tracks:

- `client_version`
- `asset_version`

Assets are hosted manually on R2. Code releases are built by GitHub Actions and deployed to Dokploy.

## Asset Hosting

Assets live on a public asset domain, for example:

```text
https://assets.geoserv.app/releases/<asset-version>/gfx/...
https://assets.geoserv.app/releases/<asset-version>/maps/...
https://assets.geoserv.app/releases/<asset-version>/pub/...
https://assets.geoserv.app/releases/<asset-version>/mfx/...
https://assets.geoserv.app/releases/<asset-version>/sfx/...
```

Recommended rule:

- never publish assets at the root
- always publish them under immutable tagged prefixes

Good examples:

- `assets-v1`
- `v0.4.0-assets`
- `2026-04-08-overhaul-a`

## Channel Manifest

Clients should read a mutable channel manifest, for example:

```text
https://assets.geoserv.app/channels/stable.json
```

Example:

```json
{
  "channel": "stable",
  "client_version": "v0.4.2",
  "asset_version": "v0.4.0-assets",
  "asset_base": "https://assets.geoserv.app/releases/v0.4.0-assets",
  "server_addr": "wss://geoserv.app/ws",
  "downloads": {
    "darwin-arm64": {
      "url": "https://geoserv.app/releases/v0.4.2/geoclient-v0.4.2-darwin-arm64.tar.gz",
      "sha256": "<archive-sha256>"
    },
    "linux-amd64": {
      "url": "https://geoserv.app/releases/v0.4.2/geoclient-v0.4.2-linux-amd64.tar.gz",
      "sha256": "<archive-sha256>"
    },
    "windows-amd64": {
      "url": "https://geoserv.app/releases/v0.4.2/geoclient-v0.4.2-windows-amd64.zip",
      "sha256": "<archive-sha256>"
    }
  }
}
```

This manifest is the control point for rollout.

## Signature Model

This repo now assumes a simple detached-signature model based on Ed25519.

Files that must be signed:

- `channels/stable.json`
- `releases/<asset-version>/manifest.json`

Each signed file must have a sibling `.sig` file containing the base64 signature:

```text
https://assets.geoserv.app/channels/stable.json
https://assets.geoserv.app/channels/stable.json.sig

https://assets.geoserv.app/releases/<asset-version>/manifest.json
https://assets.geoserv.app/releases/<asset-version>/manifest.json.sig
```

Assets themselves are not individually signed. Instead:

- `manifest.json` is signed
- each asset entry in `manifest.json` carries a SHA-256
- clients verify downloaded asset bytes against that checksum

Native binary archives are verified through the signed channel manifest:

- `stable.json` is signed
- each `downloads.<platform>.sha256` entry is trusted only because it came from the signed manifest
- the native client verifies the downloaded archive checksum before applying the update

## Signing Keys

Generate an Ed25519 keypair once:

```bash
go run ./scripts/generate_signing_key > signing-keypair.json
```

That file contains:

- `public_key`
- `private_key`

Store the private key outside the repo. The public key is what clients use to verify `stable.json` and asset manifests.

Sign a file with:

```bash
EO_UPDATE_PRIVATE_KEY='<base64-private-key>' \
go run ./scripts/sign_file --file dist/stable.json
```

That writes `dist/stable.json.sig`.

You can also sign explicitly to another path:

```bash
EO_UPDATE_PRIVATE_KEY='<base64-private-key>' \
go run ./scripts/sign_file \
  --file manifest.json \
  --out manifest.json.sig
```

The deploy workflow also generates a `dist/stable.json` file. Treat that file as the release candidate for the live channel manifest. If you keep the live channel manifest on R2, upload the generated file manually to:

```text
https://assets.geoserv.app/channels/stable.json
```

## What Each Client Does

### Web client

The web client loads `config.js`, starts WASM, and the Go runtime fetches the signed channel manifest before finalizing runtime config.

That means:

- asset-only updates can work without redeploying the web app
- changing `asset_base` in `stable.json` can move the web client to a new asset release
- unsigned or tampered channel manifests are ignored
- asset files are checked against the signed asset manifest before use

### Native client

Native release builds embed defaults for:

- `client_version`
- `server_addr`
- `asset_base`
- `update_manifest_url`
- `update_public_key`

At startup, the desktop client resolves config in this order:

- embedded release defaults
- channel manifest
- explicit environment overrides

That means the channel manifest can move native clients onto a new asset release automatically, while local overrides still win during development.

The desktop client can therefore override:

- `asset_base`
- `server_addr`

This gives native clients automatic asset/channel pickup without bundling assets locally.

Note:

- remote asset auto-pickup is implemented
- native binary self-update is implemented at startup
- the executable replacement uses the `go-selfupdate` update engine instead of shell scripts
- unsigned or tampered manifests are ignored
- asset files are checked against the signed asset manifest before use
- downloaded native archives are checked against the SHA-256 from the signed channel manifest before replacement

## Release Scenarios

### 1. Asset-only update

Use this when code stays the same but assets change.

1. Upload a new tagged asset release to R2:
   - `https://assets.geoserv.app/releases/<new-asset-version>/...`
2. Update `https://assets.geoserv.app/channels/stable.json`
   - keep `client_version` the same
   - change `asset_version`
   - change `asset_base`

You can start from the generated `dist/stable.json` and edit only the fields you want to move.

Result:

- web picks up new assets on next load
- native picks up new assets on next startup

### 2. Client-only update

Use this when binaries/web change but assets stay the same.

1. Run the GitHub Actions deploy workflow
2. Keep the same `asset_version`
3. Update `stable.json` with the new `client_version` and download URLs if desired

In practice, the generated `dist/stable.json` should already contain the new client version and download URLs. If assets did not change, publish it as-is or copy only the updated fields into your live `stable.json`.

Result:

- clients continue using the same asset set
- new web/native builds point at the existing asset release
- native desktop clients can also self-install the new binary on startup after `stable.json` moves to the new client version

### 3. Client + asset update

Use this when both code and assets change.

1. Upload new tagged asset release to R2
2. Run the deploy workflow for the new client version
3. Update `stable.json` so both:
   - `client_version`
   - `asset_version`
   move together

## GitHub Actions Workflow

The workflow does not fetch raw assets anymore.

It only:

1. runs tests
2. builds the web bundle
3. builds native archives with GoReleaser
4. assembles a Dokploy static bundle
5. syncs that bundle to Dokploy

The workflow expects asset information to be passed in, not generated from local files.

The workflow does not publish anything to R2. Asset files and the live channel manifest remain under manual control.

## GoReleaser

Native archives are built with:

- `.goreleaser.yml`

GoReleaser injects:

- `main.clientVersion`
- `main.defaultServerAddr`
- `main.defaultAssetBase`
- `main.defaultUpdateManifestURL`
- `main.defaultUpdatePublicKey`

That means native binaries ship already pointed at your production channel/asset setup.

## Dokploy Output

The deploy bundle contains code/release metadata only:

```text
dist/
  index.html
  config.js
  wasm_exec.js
  eoclient.wasm
  stable.json
  releases/<client-version>/...
```

Assets are not copied into `dist/`.

`dist/stable.json` exists so you have a generated manifest to upload to R2 when you want the channel to move.

## GitHub Variables

Repository variables expected by `.github/workflows/deploy.yml`:

- `PUBLIC_BASE_URL`
- `SERVER_ADDR`
- `ASSET_PUBLIC_BASE`
- `UPDATE_MANIFEST_URL`
- `UPDATE_PUBLIC_KEY`
- `DEPLOY_HOST`
- `DEPLOY_USER`
- `DEPLOY_PATH`

Examples:

- `PUBLIC_BASE_URL=https://geoserv.app`
- `SERVER_ADDR=wss://geoserv.app/ws`
- `ASSET_PUBLIC_BASE=https://assets.geoserv.app/releases`
- `UPDATE_MANIFEST_URL=https://assets.geoserv.app/channels/stable.json`
- `UPDATE_PUBLIC_KEY=<base64-ed25519-public-key>`

## GitHub Secrets

- `SSH_PRIVATE_KEY`
- `DOKPLOY_DEPLOY_HOOK_URL` (optional)

## Manual Asset Workflow

When assets change:

1. prepare the new tagged asset directory locally
2. upload it to R2 under:
   - `releases/<asset-version>/...`
3. generate `releases/<asset-version>/manifest.json`, for example:

```bash
go run ./scripts/release_metadata \
  --assets-dir /path/to/assets \
  --asset-version v0.4.0-assets \
  --asset-base-url https://assets.geoserv.app/releases/v0.4.0-assets \
  --asset-manifest-out manifest.json
```

4. sign it to `releases/<asset-version>/manifest.json.sig`
5. verify public access
6. update `channels/stable.json`
7. sign `channels/stable.json` to `channels/stable.json.sig`

When only the client changes:

1. run the deploy workflow
2. inspect `dist/stable.json`
3. sign `dist/stable.json`
4. upload both:
   - `channels/stable.json`
   - `channels/stable.json.sig`
5. desktop clients will update themselves on next startup

When only assets change:

1. upload the new asset tag to `releases/<asset-version>/...`
2. upload the matching signed `manifest.json` and `manifest.json.sig`
3. update `channels/stable.json`
4. sign `channels/stable.json`
5. keep `client_version` unchanged

The client repo never needs those assets in Git.

## Current Limits

Implemented now:

- web uses remote tagged assets
- desktop can use remote tagged assets
- clients can pick up channel changes for assets/server routing
- native binaries are built with GoReleaser
- desktop native binaries can self-install updates at startup
- signed channel manifests are verified
- signed asset manifests plus per-file SHA-256 are verified

Not implemented yet:

- local persistent asset cache for desktop
- automatic publishing of `stable.json` to R2
- automatic signing inside CI
- rollback UI or user-facing update prompts

The current auto-update behavior is:

- web auto-picks channel manifest changes on load
- desktop auto-picks asset and server changes from the channel manifest on startup
- desktop auto-installs a newer binary on startup when `stable.json` points to a higher semantic version for the current platform

Practical note:

- desktop auto-update compares semantic versions like `v0.4.2`
- if you publish non-semver client versions, the desktop updater is explicitly disabled for those builds
