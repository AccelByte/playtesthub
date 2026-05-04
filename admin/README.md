# playtesthub admin (Extend App UI)

React 19 + TypeScript + Vite Module Federation remote. Rendered inside the AGS Admin Portal under **Extend → My Extend Apps → App UI** (Internal Shared Cloud only — PRD §9 R11).

## Local dev

```
npm install
cp .env.local.example .env.local     # first run only; fill in VITE_AB_*
npm run codegen                      # regenerate src/playtesthubapi/ from the backend's /apidocs/api.json
npm run dev                          # Vite on :5173
```

See `docs/engineering.md` §1.2 + §7 for the full template + codegen contract.

## Scripts

- `npm run dev` — Vite dev server with `devProxyPlugin` proxying `/ext-<ns>-<app>` to AGS.
- `npm run build` — `tsc -b && vite build`. Output: `dist/`.
- `npm run codegen` — re-downloads `playtesthub.json` + regenerates `src/playtesthubapi/`. Rerun when proto HTTP annotations change.
- `npm run test` — Vitest + React Testing Library.

## Deploy

- First-time registration: `extend-helper-cli appui create --namespace $AGS_NAMESPACE --name playtesthub`.
- Upload bundle: see two-step flow below — `vite.config.ts` requires `BASE_URL` at build time, and the URL must point at the **parent-namespace host**, not the game-namespace host the upload CLI prints.

### Two-step build + upload (Module Federation publicPath)

`vite.config.ts` reads `process.env.BASE_URL` only in production mode and bakes it into `mf-manifest.json` as `publicPath`. The Admin Portal host loads `remoteEntry.js` from that exact URL — if it's wrong (or empty), the federation import 404s with `Failed to fetch dynamically imported module: …/remoteEntry.js`.

CSM serves AppUI assets only from the **parent namespace's host** (`<parent>.internal.gamingservices.accelbyte.io`), not the game namespace's host (`<parent>-<game>.internal…`). Hitting the latter returns `404 data not found: subdomain mismatch`. The dev `extend-helper-cli appui upload` reports the wrong host in its "Asset Base URL" log line — ignore it; use the parent host.

Pin a build version up front so the BASE_URL path matches the upload path. CSM rejects re-upload of an existing version (`GeneralError(20024): version already exists for this app UI`) — pick a fresh `$VERSION` for every upload; bump it on retry.

```bash
VERSION=m2pool02      # any short identifier; must be unused — bump on every retry

# Substitute: AB_PARENT = parent namespace ID (no -<game> suffix)
#             AB_NAMESPACE = game namespace ID
BASE_URL="https://${AB_PARENT}.internal.gamingservices.accelbyte.io/csm/v1/admin/namespaces/${AB_NAMESPACE}/files/app-ui/playtesthub/${VERSION}/" \
  npm run build

extend-helper-cli appui upload \
  --namespace "$AB_NAMESPACE" --name playtesthub \
  --build-version "$VERSION" --no-build
```

Local sanity check before reloading:

```bash
grep -o '"publicPath":"[^"]*"' dist/mf-manifest.json    # must match the BASE_URL above (parent host)
```

The only authoritative deploy check is **browser DevTools → Network** in the live Admin Portal: hard-reload (Ctrl/Cmd+Shift+R), filter for `mf-manifest.json` + `remoteEntry.js`, and confirm both return 200. Do **not** rely on `curl` against the CSM file URL — a client_credentials / IAM-admin Bearer has different CSM visibility than the Admin Portal session cookie, and can return 404 on URLs the Admin Portal serves fine (or 200 on URLs the Admin Portal can't reach). Cookie-cross-origin auth is what makes the bundle load, and only the browser exercises it.
