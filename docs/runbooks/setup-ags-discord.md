# Setup — AGS + Discord for player login

Step-by-step setup a self-hosting playtesthub operator must complete on a fresh AGS tenant before the player Discord login works. Written assuming zero prior AGS knowledge.

This runbook is **prescriptive** ("do this, in this order, on these screens"). The companion descriptive doc — what the working flow looks like end-to-end, byte-exact URL shapes, verified payload, AGS source-code references — is [`discord-login.md`](discord-login.md). The architectural rationale (why we use the platform-token grant rather than auth-code federation) lives in `docs/engineering.md` § "Discord federation via platform-token grant" and STATUS.md M1 phases 9.2–9.4.

If you only read one section of this file, read [§ Three URLs that must agree byte-for-byte](#three-urls-that-must-agree-byte-for-byte). That single constraint is the highest-leverage piece of context for getting login working on a new tenant.

## Prerequisites

- AGS Shared Cloud or self-hosted AGS tenant. You own a game namespace under it.
- AGS Admin Portal access on the game namespace with `ADMIN:NAMESPACE:{namespace}:CLIENT [CRUD]`.
- A Discord application you control. Free tier is fine.
- A deployed (or local) playtesthub backend you can set env vars on, plus the player Vite bundle hosted somewhere with a stable origin.

## Steps

### 1. Discord developer portal

`https://discord.com/developers/applications` → **New Application** → name it (e.g. `Playtesthub - Acme Studios`).

Then **OAuth2** → **Redirects** → add one entry per environment that signs in users:

- Local dev: `http://localhost:5173/callback`
- Production: `https://<your-player-origin>/callback`
- Any preview/staging: same shape

Discord matches the redirect URI **byte-for-byte** including scheme, port, and absence of trailing slash. If it's not on this allowlist, `discord.com/oauth2/authorize` errors with **"Invalid OAuth2 redirect_uri"** rendered on Discord's domain — the player never lands back in the app.

Capture two values from the **OAuth2** page:

- **Client ID** — public; goes into `player/public/config.json` as `discordClientId`.
- **Client Secret** — confidential; pasted into AGS Admin Portal in step 2. **Never** put this in playtesthub config or source.

### 2. AGS Admin Portal — Discord platform credential

In Admin Portal, navigate to **{your namespace} → Login Methods → Platforms → Discord** (URL shape: `https://<ags-host>/admin/namespaces/<namespace>/login-methods/platforms/discord`).

Fill in:

| Field | Value | Why |
| --- | --- | --- |
| Client ID | The Discord **Client ID** from step 1. | AGS uses this when calling `discord.com/api/oauth2/token` on the backend's behalf. |
| Client Secret | The Discord **Client Secret** from step 1. | Same. |
| RedirectUri | **Byte-exact** `${PLAYER_ORIGIN}/callback` — the same string the player sends to `discord.com/oauth2/authorize`. | See [§ Three URLs that must agree byte-for-byte](#three-urls-that-must-agree-byte-for-byte). The AGS docs default `https://<your-ags-host>/iam/v3/platforms/discord/authenticate` is **wrong** for this flow and produces a redirect_uri mismatch on every login. |
| IsActive | `true` | If `false`, AGS rejects platform-token grants for Discord before Discord is ever called. |

Save. Verify with the public probe:

```sh
curl -s "${AGS_BASE_URL}/iam/v3/public/namespaces/${AGS_NAMESPACE}/platforms/clients/active" | jq '.[] | select(.PlatformID=="discord")'
```

Expected: a row with `IsActive: true` and your configured `RedirectUri`. If Discord doesn't appear or `IsActive: false`, the toggle didn't persist — fix before continuing.

### 3. AGS confidential IAM client

The backend uses **one** confidential AGS IAM client for two purposes:

1. Validating IAM JWTs on admin/player RPCs (handled by `pkg/iam`).
2. Calling `POST /iam/v3/oauth/platforms/discord/token` with HTTP Basic auth (handled by `pkg/service/discord_exchange.go`).

Create or reuse a confidential client with:

- Client type: **Confidential**.
- Permissions: those required by the existing playtesthub deploy guide for IAM JWT validation, plus the Discord-grant permission. If AGS rejects with `unauthorized_client` during a real Discord login attempt, this is what's missing. Assign `NAMESPACE:{namespace}:USER:LOGIN [CREATE]` (or whatever your AGS role catalogue calls the equivalent) and retry.

Capture **Client ID** + **Client Secret** — these become `AGS_IAM_CLIENT_ID` / `AGS_IAM_CLIENT_SECRET` in the backend env. There is no separate "player IAM client" in this flow; the phase-9.1-era public IAM client (PKCE-only, no secret) is no longer used.

### 4. Backend env vars

Set on the deployed backend (or in `.env` for local docker-compose):

```sh
AGS_BASE_URL=https://<your-ags-host>
AGS_NAMESPACE=<your-game-namespace>
AGS_IAM_CLIENT_ID=<from step 3>
AGS_IAM_CLIENT_SECRET=<from step 3>
DISCORD_BOT_TOKEN=<bot token, separate from OAuth app — used by pkg/discord for handle lookup at signup per PRD §10 M1>

# Required when the player is hosted off-origin (GitHub Pages,
# Vercel, custom domain). Empty here means cross-origin preflights
# fail with 501. See docs/runbooks/deploy-player-pages.md § 3.
CORS_ALLOWED_ORIGINS=https://<your-player-origin>
```

`.env.template` is the canonical list of required variables.

### 5. Player config

`player/public/config.json` (committed for your deploy; one file per environment):

```json
{
  "grpcGatewayUrl": "https://<your-deployed-backend>/<base-path>",
  "iamBaseUrl":     "https://<your-ags-host>",
  "discordClientId": "<Discord Client ID from step 1>",
  "discordInviteUrl": "https://discord.gg/<your-server-invite>"
}
```

`discordClientId` is the **Discord** OAuth client ID, not an AGS IAM client. `iamBaseUrl` is no longer used by the Discord exchange path — it's wired through for SDK / observability code that still references it.

`discordInviteUrl` is **optional but strongly recommended**. When set, the pending page renders a "Join our Discord while you wait" link so applicants land in your studio's server before approval. This is operationally required for outbound DMs to land — see § Discord bot + server below. For Pages deploys, the `pages.yml` workflow reads `PLAYER_DISCORD_INVITE_URL` from Repo Variables and emits this field automatically.

### 6. CLI loopback origin (`pth auth login --discord`)

The `pth` CLI's `auth login --discord` flow (cli.md §7.1) runs Discord OAuth in the operator's browser, receives the authorization code on a **loopback HTTP listener**, and POSTs the code to the backend's `Player.ExchangeDiscordCode` RPC. Architecturally identical to the player flow — the same "three URLs that agree byte-for-byte" constraint applies — but the canonical origin shifts from `${PLAYER_ORIGIN}` to a fixed loopback URL.

Default CLI loopback origin: `http://127.0.0.1:14565`. Override via `PTH_DISCORD_LOOPBACK_PORT`. The port is **fixed**, not ephemeral, because Discord's redirect-URI allowlist is byte-exact — see § Three URLs that must agree byte-for-byte.

To enable CLI Discord login, repeat steps 1 and 2 with the CLI value:

1. **Discord developer portal → OAuth2 → Redirects** — add `http://127.0.0.1:14565/callback` alongside the existing player entries.
2. **AGS Admin Portal → Login Methods → Platforms → Discord → RedirectUri** — set this to `http://127.0.0.1:14565/callback`.

Step 2 is the operationally inconvenient one. **AGS allows exactly one `RedirectUri` per Discord platform credential per namespace.** That implies a hard choice per AGS namespace:

| You want | Solution |
| --- | --- |
| Only player Discord login working | Set RedirectUri to `${PLAYER_ORIGIN}/callback`. Don't run `pth auth login --discord`; use `pth auth login --password` against a `pth user create`'d native user instead. |
| Only CLI Discord login working | Set RedirectUri to `http://127.0.0.1:14565/callback`. The player web app's Discord login will fail until reverted. Useful for solo CLI-driven dev. |
| Both working in the same namespace | Not supported by a single AGS Discord credential. Options: (a) two AGS namespaces, each with its own Discord platform credential, or (b) two Discord OAuth applications — one for player, one for CLI — wired into separate AGS configs. Most self-hosted dev setups pick (a) by convention: a personal `dev-cli` namespace for CLI smoke + `dev-player` for browser testing. |

The e2e test suite (M1 phase 11) does **not** need this — it uses password-grant native users (cli.md §7.4). CLI Discord login is purely a developer affordance.

CLI env vars required for `pth auth login --discord`:

```sh
PTH_DISCORD_CLIENT_ID=<Discord Client ID from step 1>
PTH_DISCORD_LOOPBACK_PORT=14565   # default; override only if 14565 is occupied locally
PTH_BACKEND_REST_URL=https://<your-deployed-backend>/<base-path>  # HTTPS gateway base; the exchange POST goes here, not the gRPC --addr
```

Verification: after registering the redirect URI in both places, run `pth auth login --discord --no-browser --dry-run`. CLI prints the authorize URL + listener address + exchange URL as JSON and exits 0 — no network call. Then run `pth auth login --discord` for real, complete Discord login in the browser, and verify `pth auth whoami` returns the federated `userId` (matches the AGS JWT `sub` after the platform-token grant auto-creates the Justice account). Failure modes mirror the player flow — see § Common failure modes.

### 7. Discord bot + server (required for DM delivery)

Code grants are delivered by the backend's DM worker (`pkg/dmqueue` + `pkg/discord`) via the Discord REST API. **Discord blocks bot DMs to any user who does not share at least one guild with the bot** — the API returns HTTP 403 with code `50278` ("Cannot send messages to this user due to having no mutual guilds"). The signup flow proceeds, but the DM never arrives; `applicant.last_dm_status='failed'` and `last_dm_error` carries the 403.

There is no workaround at the API layer. The bot must be a member of a Discord server that approved playtesters also join, **before approval**. The recommended shape:

1. **Create a Discord server** for the playtest (or reuse an existing studio server). Free; one server can host many playtest cohorts via channels or roles.
2. **Create a bot user under the same Discord application from step 1.**
   - Discord developer portal → your application → **Bot** → **Add Bot**.
   - Copy the bot token. This is `DISCORD_BOT_TOKEN` in the backend env. The OAuth Client Secret from step 1 and the bot token are **two different secrets**; do not interchange them.
   - Privileged intents are not required — playtesthub uses only `POST /users/@me/channels` + `POST /channels/{id}/messages`, both of which work with a bare bot token.
3. **Invite the bot into your server.**
   - Discord developer portal → **OAuth2** → **URL Generator** → scope `bot`, no permissions required (`permissions=0`).
   - Open the generated URL while logged into a Discord account that owns the server, pick the server, authorize.
   - Verify with `curl -H "Authorization: Bot $DISCORD_BOT_TOKEN" https://discord.com/api/v10/users/@me/guilds` — the response must be a non-empty array. An empty `[]` is the signature failure mode.
4. **Get applicants into the same server.** Surface the invite URL to applicants while they wait for approval:
   - Set the `PLAYER_DISCORD_INVITE_URL` repo Variable (Pages deploy) or add `discordInviteUrl` to `player/public/config.json` directly. The pending page renders a "Join our Discord while you wait" link when this value is set.
   - Use a [permanent invite link](https://support.discord.com/hc/en-us/articles/208866998-Invites-101) so the URL stays stable across config bumps.
   - Recipient-side: applicants must have **User Settings → Privacy & Safety → Allow direct messages from server members** enabled (Discord's default for new accounts; only matters for users who toggled it off).
5. **Optional — embed a deep link to the pending page in the DM body.** Set `PLAYER_BASE_URL` on the backend (e.g. `https://anggorodewanto.github.io/playtesthub`); the DM becomes:

   ```
   You're approved for "Acme Closed Beta". View your code: https://anggorodewanto.github.io/playtesthub/#/playtest/acme-beta/pending
   ```

   Discord renders the bare URL as a tappable link. The route requires the recipient to be Discord-authed on the player domain — sessionStorage from signup does not survive across domains, so the first tap may bounce through Discord login (one Approve click, since they are already federated) before landing on Pending. When `PLAYER_BASE_URL` is empty the DM falls back to the legacy non-clickable copy.

6. **Verify end-to-end** before opening signups:
   - Sign up with a test Discord account that is a member of the server. Approve the applicant from the admin UI. The DM should arrive within seconds; `applicant.last_dm_status='sent'`.
   - If `last_dm_status='failed'` with `last_dm_error` containing `code 50278`, the test account is not in the server — add it and retry via `pth applicant retry-dm` or the admin UI's RetryDM button.

Operating note: a single bot token can DM users across all guilds it is a member of, so multi-server studios can run one bot per Discord application across as many servers as they have playtests, as long as each cohort joins at least one shared server with the bot.

## Three URLs that must agree byte-for-byte

The `redirect_uri` value flows through three independent systems. All three must be identical strings — character for character, including scheme, port, and absence of trailing slash. Get all three byte-equal and the flow works; miss any one and Discord rejects with a specific error documented under [§ Common failure modes](#common-failure-modes).

| Where it lives | Value | Why it matters |
| --- | --- | --- |
| Discord developer portal → OAuth2 → Redirects | `${PLAYER_ORIGIN}/callback` | Discord rejects `/oauth2/authorize` with **"Invalid OAuth2 redirect_uri"** if the player sends a value not on this allowlist. |
| Player's call to `discord.com/oauth2/authorize` (`buildDiscordAuthorizeUrl` in `player/src/lib/auth.ts`, fed by `window.location.origin`) | `${PLAYER_ORIGIN}/callback` | Discord stores this value alongside the issued auth code. |
| AGS Admin Portal → Login Methods → Platforms → Discord → **RedirectUri** | `${PLAYER_ORIGIN}/callback` | When AGS POSTs to `discord.com/api/oauth2/token` to redeem the code, it forwards this configured value. Discord byte-compares against the value the player sent at /authorize. Mismatch → `400 invalid_grant: Invalid "redirect_uri" in request.` |

The third row is the load-bearing trap. AGS's platform-token grant **does not honor a caller-supplied `redirect_uri`** form-body parameter — see [`discord-login.md` § Why AGS ignores our form-body redirect_uri](discord-login.md#why-ags-ignores-our-form-body-redirect_uri) for the verified AGS source path.

**Implication**: one AGS Discord platform credential ⇒ one canonical `${PLAYER_ORIGIN}` per AGS tenant. Dev (`http://localhost:5173/callback`) and prod cannot share unless they share an origin. If you need both dev and prod, you need two AGS namespaces with their own Discord platform credentials — or override the AGS RedirectUri value when switching environments.

## Verification ladder

Run in order. Don't skip; each step builds on the last.

1. **Smoke harness against the deployed backend** — `scripts/smoke/cloud.sh` exits 0. Probes the surface-level wiring (RPC routed, auth interceptor accepts cookies, etc.).
2. **Smoke harness with a forced bogus exchange** — `scripts/smoke/cloud.sh` posts an obviously-fake Discord code to `/v1/player/discord/exchange` and asserts a 400 with `invalid_grant` somewhere in the body. This validates AGS Basic-auth + the AGS-wraps-Discord-invalid_grant detection in `mapAGSExchangeError` — even before any real user exists. If this fails, your `AGS_IAM_CLIENT_ID` / `AGS_IAM_CLIENT_SECRET` are wrong, or the AGS IAM client lacks the Discord-grant permission.
3. **Manual browser smoke** — open the player at `${PLAYER_ORIGIN}/#/playtest/<seeded-slug>`, click Sign up. Discord consent screen appears (Discord's domain). Approve. Lands back on `${PLAYER_ORIGIN}/callback` then bounces to `/#/signup`. Submit the platforms form. Lands on `/#/pending`. The applicant row exists in Postgres with `status=PENDING` and `userId` matching the AGS JWT `sub`.

If any step fails, see [§ Common failure modes](#common-failure-modes) before changing config — most failure modes have specific symptoms that identify the misconfiguration directly.

## Common failure modes

Each row is a 9.4 reproduction. Byte-exact error strings live in [`discord-login.md` § Failure modes seen during phase 9.4 verification](discord-login.md#failure-modes-seen-during-phase-94-verification).

| Symptom | Root cause | Fix |
| --- | --- | --- |
| **`Invalid OAuth2 redirect_uri`** rendered on `discord.com/oauth2/authorize`. No callback fires; the player never lands back at the app. | `${PLAYER_ORIGIN}/callback` is not on the Discord developer portal's OAuth2 → Redirects allowlist. | Add it. Discord matches byte-exactly. |
| **`POST /v1/player/discord/exchange → 400`** with body containing `discord.com/api/oauth2/token 400 {"error": "invalid_grant", "error_description": "Invalid \"redirect_uri\" in request."}`. | AGS Admin Portal Discord `RedirectUri` ≠ player's `${PLAYER_ORIGIN}/callback`. | Set them byte-equal. The AGS-docs default value is wrong for this flow — see step 2. |
| **`POST /v1/player/discord/exchange → 400`** with body containing `discord.com/api/oauth2/token 400 {"error": "invalid_grant", "error_description": "Invalid \"code\" in request."}`. | The Discord code is bogus / already used / expired. The smoke probe deliberately produces this. | Real users: retry — fresh OAuth roundtrip. Smoke probe: this is the success signal. |
| **AGS Discord platform `IsActive=false`** — AGS rejects the grant before Discord is called. | Toggle didn't persist in Admin Portal. | Verify with the `GET /iam/v3/public/namespaces/{namespace}/platforms/clients/active` probe in step 2. If Discord doesn't appear with `IsActive: true`, fix the toggle before retrying. |
| **AGS returns `unauthorized_client`**. | Confidential IAM client lacks the Discord-grant permission. | Assign the equivalent of `NAMESPACE:{namespace}:USER:LOGIN [CREATE]` per your AGS role catalogue. |
| **First `POST /v1/player/discord/exchange` of a session occasionally returns HTTP 503**, but replaying the same code via `curl` seconds later returns 200. | Suspected AGS Discord-call latency on cold path or vite dev-proxy short timeout. Tracked as a STATUS.md follow-up; not a setup bug. | Retry once. If it persists across retries, escalate. |
| **`Applicant.discordHandle=""`** on a fresh signup. | Discord bot token unset, or AGS rate-limited the bot. PRD §10 M1 falls back to raw Discord ID; an empty value points at the bot token, not setup. | Set `DISCORD_BOT_TOKEN`. Bot setup is § 7 above. |
| **`applicant.last_dm_error` contains `unexpected status 403` and `code 50278` ("Cannot send messages to this user due to having no mutual guilds")** after approval. | The bot does not share a Discord server with the recipient. Discord refuses bot DMs in this case at the API layer. | Add the bot to a server, get applicants to join it before approval. Walk-through in § 7 above. Verify with `curl -H "Authorization: Bot $DISCORD_BOT_TOKEN" https://discord.com/api/v10/users/@me/guilds` — must be non-empty. |
| **`applicant.last_dm_error` is `missing_recipient`** after approval, with `applicant.discord_user_id` empty. | Applicant did not federate via Discord at signup (e.g., signed up with a non-Discord IAM identity), so no Discord snowflake was ever stored. The DM worker has nowhere to send. | Expected outcome per [`docs/dm-queue.md`](../dm-queue.md). To require Discord-only signup, surface this constraint in your studio's signup messaging — the backend does not enforce it. |
| **`Applicant.discordHandle=""` and `Applicant.platforms=[]` in the `GET /applicant` response**. | **Not a bug.** `discordHandle` and `platforms` are admin-only fields per `docs/schema.md` L88. The player-visible response strips them; the DB row has the data. | Verify the actual DB row via the admin API or a direct SQL query. |

For wire-level error contracts (which `ExchangeDiscordCode` errors map to which gRPC codes), see [`docs/errors.md`](../errors.md).

## Cross-references

- [`deploy-player-pages.md`](deploy-player-pages.md) — companion runbook for shipping the player on GitHub Pages. Adds the Pages-shaped `${PLAYER_ORIGIN}` value (= `https://<owner>.github.io`) and the corresponding callback URL (= `https://<owner>.github.io/<repo>/callback`) to the byte-exact rule above.
- [`discord-login.md`](discord-login.md) — descriptive companion: verified URL shapes, AGS source-code references, the verified successful payload.
- [`docs/engineering.md`](../engineering.md) § "Discord federation via platform-token grant" — flow table + architectural rationale (why platform-token grant, not auth-code federation).
- [`docs/PRD.md`](../PRD.md) §5.2 — Discord login as a player requirement.
- [`docs/errors.md`](../errors.md) — byte-exact wire contract for `ExchangeDiscordCode` errors.
- STATUS.md M1 phase 9.3 outcome — architectural rationale (why we ditched the auth-code path).
- STATUS.md M1 phase 9.4 outcome — the `mapAGSExchangeError` patch + AGS-RedirectUri trap.

## Out of scope

- AGS tenant provisioning. Assumed pre-existing.
- Discord-server moderation, channel layout, role automation. § 7 covers only what playtesthub needs: a bot user, an invite for that bot, and a server-invite link applicants can join.
- Non-Discord platform login. The architecture is generic enough to extend, but only Discord is wired today (PRD §5.2).
