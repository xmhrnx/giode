# Contributing

Giode is young and contributions are welcome: bug reports, fixes, components, docs and
examples.

## Reporting a bug

Open an [issue](https://github.com/xmhrnx/giode/issues) with:

- what you did, what you expected and what happened;
- the platform (OS and version, phone model), Go version and Giode version;
- the smallest program that shows it, if you can.

## Working on the code

```sh
git clone https://github.com/xmhrnx/giode
cd giode
go test ./...
```

- **Style.** `gofmt`, doc comments on everything exported, and comments that say why
  rather than what.
- **Tests.** Components are tested by laying them out on an `op.Ops` with a fake
  `input.Router` and checking what they do; see any `*_test.go` under `components/`.
- **Platform code.** Files are split by build tags (`_android.go`, `_darwin.go`,
  `_windows.go`, ...). Say in the pull request which platforms you could test on.
- **Re-exports.** The root package's short names are generated from package `ui`: add a
  name there and run `go generate` in the repository root.

## Writing docs

The docs are the Markdown files in [`docs/`](https://github.com/xmhrnx/giode/tree/main/docs);
the website renders them. Run the site locally with:

```sh
cd web
npm install
npm run dev
```

Add a new page to `NAV` in `web/src/nav.js`.

## Adding your app to the examples

Built something with Giode? Open a pull request adding it to `web/src/projects.js`, with a
square icon in `web/public/examples/<name>/`.
