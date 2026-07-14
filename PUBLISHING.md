# Publishing & releasing yahh

How releases and the Homebrew formula are wired. Same model as the other
Positronico projects (snapem, claudeq): **the tap is updated manually — no
PAT or cross-repo automation exists anywhere.**

## The two repos

| Repo | Holds | URL |
|---|---|---|
| `Positronico/yahh` | this project (Go source, CI, goreleaser config) | https://github.com/Positronico/yahh |
| `Positronico/homebrew-tap` | the Homebrew formula (`Formula/yahh.rb`) | `brew install Positronico/tap/yahh` |

## CI/CD (in `Positronico/yahh`)

- **`.github/workflows/ci.yml`** — every push/PR: `go vet`, unit tests,
  build, and the interactive-shell e2e suites on ubuntu + macos.
- **`.github/workflows/release.yml`** — on a `vX.Y.Z` tag: goreleaser builds
  darwin/linux × amd64/arm64 tarballs and publishes a GitHub Release with
  `checksums.txt`. Uses only the built-in `GITHUB_TOKEN` (which cannot push
  to the tap — hence the manual formula step below).

## Cutting a release

1. Make sure `main` is green, then tag and push:
   ```bash
   git tag vX.Y.Z && git push --tags
   ```
   `release.yml` publishes the binaries and `checksums.txt`.

2. **Homebrew:** regenerate the formula (fills the version and all four
   sha256s from the release's `checksums.txt`) and push it to the tap:
   ```bash
   packaging/update-formula.sh vX.Y.Z > ../homebrew-tap/Formula/yahh.rb
   cd ../homebrew-tap
   git add Formula/yahh.rb && git commit -m "yahh X.Y.Z" && git push
   ```

3. Smoke-test:
   ```bash
   brew update && brew install Positronico/tap/yahh   # or: brew upgrade yahh
   yahh version
   ```

To automate step 2 later: uncomment the `brews:` section in
`.goreleaser.yaml` and add a PAT with write access to the tap as the
`HOMEBREW_TAP_GITHUB_TOKEN` secret on `Positronico/yahh` (a fine-grained
PAT scoped to only `Positronico/homebrew-tap` with Contents: read/write
is enough).
