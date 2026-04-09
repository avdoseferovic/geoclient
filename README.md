# geoclient

`geoclient` is the GeoServ game client, built in Go with a shared codebase for both the native Ebiten desktop app and the web/WASM client. The repo contains the client runtime, rendering, UI, network protocol handling, and supporting release scripts for shipping the same game across desktop and browser targets.

## Local Development

The client expects local asset directories in the repo root such as `gfx/`, `maps/`, and `pub/`. Those are the runtime asset sources used for local development; the old `/data` directory is not part of the current client path. For desktop development it reads those files directly from disk and, unless overridden, connects to `ws://127.0.0.1:8078`.

Use Go 1.26 or newer, then start the native client with:

```bash
go run ./cmd/eoclient
```

To point the desktop client at another server or remote asset release, set:

```bash
EO_SERVER_ADDR=wss://example.com/ws
EO_ASSET_BASE=https://assets.example.com/releases/dev-assets
go run ./cmd/eoclient
```

For web development, build the WASM bundle and serve it with the included dev server:

```bash
./scripts/build-web.sh
go run ./cmd/webclient
```

That serves the app on `http://127.0.0.1:8090`, serves repo assets under `/assets/`, and by default connects the browser client to `ws://<host>:8078`. You can also override the web runtime at load time with query parameters such as `?serverAddr=ws://127.0.0.1:8078` and `?assetBase=https://assets.example.com/releases/dev-assets`.

## Deployment

Deployment is split across code and assets so runtime assets never need to live in Git. Versioned asset bundles are published to R2 under immutable release prefixes, while application releases are built by GitHub Actions and deployed to Dokploy.

Clients roll forward through a signed channel manifest that points to the current client version, asset version, asset base URL, server address, and platform download metadata. Asset manifests and channel manifests are signed with Ed25519, and clients verify published SHA-256 checksums before trusting downloaded archives or assets.
