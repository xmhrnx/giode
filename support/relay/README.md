# Support relay

A [Cloudflare Worker](https://developers.cloudflare.com/workers/) that turns the problem
reports Giode apps send with [`giode/support`](../../docs/support.md) into GitHub issues.
Users need no GitHub account, and the GitHub token stays in the Worker instead of
shipping in every copy of the app. One relay serves all your apps. Cloudflare's free plan
covers it: 100,000 requests a day.

## What it does

- `POST /report`: checks the report, rate-limits each network (5 an hour and 20 a day by
  default), and opens an issue in the app's repository, labelled `user report`, plus
  `crash` when a crash is attached. Mentions (`@name`) and issue links (`#12`) in the
  user's text are neutralised, so a report can't ping anyone.
- The reporter's email, if they left one, is **not** put in the public issue. It's kept in
  the Worker's KV store for a year, and you read it with
  `GET /contact?repo=owner/repo&issue=12` and `Authorization: Bearer <ADMIN_TOKEN>`.

## Deploy

You need Node.js and a free Cloudflare account.

1. Create a GitHub token at <https://github.com/settings/personal-access-tokens/new>:
   fine-grained, with access to the apps' repositories only, and the **Issues: Read and
   write** permission.
2. In `wrangler.jsonc`, list each app in `APPS` under the name it passes as
   `support.Options.App`, with its repository.
3. In this folder:

   ```sh
   npx wrangler@latest login
   npx wrangler@latest deploy                  # creates the KV namespace too
   npx wrangler@latest secret put GITHUB_TOKEN # the token from step 1
   npx wrangler@latest secret put ADMIN_TOKEN  # any long random string, for /contact
   npx wrangler@latest secret put KEY          # optional: the key your apps send
   ```

   `deploy` prints the address, such as `https://giode-support.<you>.workers.dev`. The
   apps' `ReportURL` is that address plus `/report`. Secrets take effect at once; changes
   to `wrangler.jsonc` need another `deploy`.

`wrangler.jsonc` is the only configuration file: don't add a `wrangler.json` or
`wrangler.toml` next to it (Wrangler reads just one, and a snippet pasted into a new
file has no `name` or `main`, so nothing deploys). The KV namespace has no `id` there:
`deploy` creates it. If your Wrangler asks for one instead, run
`npx wrangler@latest kv namespace create SUPPORT`, let it add the id to `wrangler.jsonc`,
and deploy again.

## Check it

Open the address in a browser. It shows what's set, without the secrets:

```json
{"ok":true,"apps":["sonora","easysend"],"github_token":true,"admin_token":true,"key_required":false,"kv":true}
```

`ok` is false while `apps` is empty, `GITHUB_TOKEN` is missing or the KV namespace isn't
bound. Then send a test report:

```sh
curl -X POST https://giode-support.<you>.workers.dev/report \
  -H 'Content-Type: application/json' \
  -d '{"app":"sonora","title":"Test report","description":"Sent with curl."}'
```

Every answer is JSON, and a failure's `error` says what went wrong, with GitHub's own
message when GitHub refused:

| Error | Fix |
| --- | --- |
| `nothing at /...` (404) | `ReportURL` must end in `/report`. |
| `unknown app "x": the relay's APPS lists ...` | Add the app to `APPS` and deploy. |
| `the relay has no GITHUB_TOKEN secret` | `npx wrangler secret put GITHUB_TOKEN`. |
| `GitHub rejected the relay's GITHUB_TOKEN (401)` | The token is wrong or expired: make a new one. |
| `GitHub can't find owner/repo (404)` | Fix the name in `APPS`, or add the repository to the token's access. |
| `GitHub refused to open an issue in owner/repo (403)` | Give the token **Issues: Read and write**. |

A 404 or 401 the app shows without such a message didn't come from the relay: nothing is
deployed at that address (or something in front of it, such as Cloudflare Access, turned
the request away). Each request is logged with its error in Workers Logs (the dashboard's
**Workers & Pages → giode-support → Logs**), or live with `npx wrangler tail`.

`npm test` runs the relay's tests (Node 20 or later, nothing to install).

## Settings

| Name | Kind | Meaning |
| --- | --- | --- |
| `APPS` | var | App name to `owner/repo`. Reports from other apps are refused. |
| `GITHUB_TOKEN` | secret | Opens the issues. |
| `ADMIN_TOKEN` | secret | Reads reporters' emails at `/contact`. |
| `KEY` | secret, optional | When set, reports must carry it (`support.Options.Key`). It ships inside the apps, so it stops casual spam, not someone who takes an app apart. |
| `LABELS` | var | Comma-separated labels for every report. Default `user report`. |
| `PER_HOUR`, `PER_DAY` | var | Reports one network may send. Default 5 and 20. |
| `SUPPORT` | KV binding | Rate-limit counters and emails. |
