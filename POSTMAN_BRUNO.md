# Testing the API with Postman or Bruno

Both collections cover the full REST surface — health/readiness/metrics, auth, post CRUD, bulk create, JSON/CSV export, character-frequency analytics, async reanalysis, MinIO export listing/download, ML sentiment classification, admin status, and a full set of ABAC role-boundary checks (viewer/editor/admin). They're generated from the same request set, so picking one tool over the other doesn't mean missing coverage.

| | Postman | Bruno |
|---|---|---|
| Files | `postman/Post-Analyzer.postman_collection.json` + `postman/Post-Analyzer.postman_environment.json` | `bruno/` (one `.bru` file per request, folders mirror Postman's) |
| Storage | JSON blob | Plain text, git-diff-friendly |
| Requires an account? | No (Postman app itself doesn't require sign-in to run a local collection) | No — Bruno is fully offline, no cloud sync at all |
| CI | `make postman-run` (newman) | not wired into CI — Bruno's CLI (`bru`) can run it the same way (see below) if you want to |

## Postman

1. Open Postman → **Import** → select both `postman/Post-Analyzer.postman_collection.json` and `postman/Post-Analyzer.postman_environment.json`.
2. Select the **Post Analyzer - Local** environment (top-right dropdown). Defaults to `baseUrl = http://localhost:8080` — change it if your gateway is reachable elsewhere (e.g. `http://localhost` if you're going through the nginx edge from `docker-compose.yml`).
3. Run **Auth → Login (admin)** first — its test script captures the JWT into the `token` environment variable, which every other request's Authorization header (`Bearer {{token}}`, set at the collection level) then uses automatically.
4. From there, run requests individually, or use **Collection Runner** to run the whole collection top to bottom — later requests reuse `{{postId}}`, `{{exportKey}}`, etc. captured by earlier ones.

Example: create a post, then fetch it back, entirely via the collection's variables —

```
Auth / Login (admin)              → captures {{token}}
Posts / Create Post               → captures {{postId}}
Posts / Get Post                  → GET /api/v1/posts/{{postId}}, uses {{token}} automatically
```

Command line, via [newman](https://github.com/postmanlabs/newman) (what `make postman-run` runs):

```bash
npx newman run postman/Post-Analyzer.postman_collection.json \
  -e postman/Post-Analyzer.postman_environment.json
```

## Bruno

[Bruno](https://www.usebruno.com/) stores each request as a plain-text `.bru` file instead of one large JSON blob — diffs in PRs show exactly which request changed, and nothing is stored in the cloud.

1. Open Bruno → **Open Collection** → select the `bruno/` folder.
2. Select the **Local** environment (matches Postman's defaults: `baseUrl = http://localhost:8080`).
3. Same flow as Postman: run **Auth → Login (admin)** first (its `script:post-response` block calls `bru.setVar("token", ...)`), then anything else — the collection-level `auth { mode: bearer }` (in `collection.bru`) means every request sends `Authorization: Bearer {{token}}` unless it explicitly sets `auth { mode: none }` (health checks, login itself).

Command line, via the [Bruno CLI](https://docs.usebruno.com/bru-cli/overview):

```bash
npx @usebruno/cli run bruno --env Local
```

## Example requests (both tools send the same thing)

**Login:**

```
POST {{baseUrl}}/api/v1/auth/login
Content-Type: application/json

{"username": "admin", "password": "admin123"}
```

**Create a post (needs the token from Login):**

```
POST {{baseUrl}}/api/v1/posts
Authorization: Bearer {{token}}
Content-Type: application/json

{"userId": 1, "title": "Hello", "body": "Created via Postman/Bruno"}
```

**Classify sentiment:**

```
POST {{baseUrl}}/api/v1/ml/sentiment
Authorization: Bearer {{token}}
Content-Type: application/json

{"text": "I love this, fantastic work"}
```

**ABAC in action** — log in as `viewer`/`viewer123` and the same create-post request above now returns `403` with a specific reason (`"role 'viewer' has no write permission on resource 'post'"` or similar — see the [ABAC section of ARCHITECTURE.md](./ARCHITECTURE.md#abac-policy-evaluation)), not a generic error. The **ABAC Role Checks** folder in both collections walks through this exact scenario for all three roles.

## Keeping both in sync

The Bruno collection is generated from the Postman collection (same request set, folder structure, and test assertions translated to Bruno's `test()`/`expect()` syntax) rather than hand-maintained separately — if you add or change a request, update the Postman collection and regenerate `bruno/` from it so the two never drift apart.
