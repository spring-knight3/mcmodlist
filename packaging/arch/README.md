# mcmodlist Arch packaging

Two PKGBUILDs live here:

- `PKGBUILD` — split release package: `mcmodlist` (CLI) and
  `mcmodlist-gui` (Qt 6 GUI **plus** `mcmodlist-engine`, which the GUI
  locates next to its own executable). The CLI has no GUI dependency;
  `mcmodlist` lists `mcmodlist-gui` in `optdepends`.
- `PKGBUILD-git` — `mcmodlist-git`, a single package building all three
  binaries from the latest git HEAD (`pkgver()` derives from
  `git describe`).

> **NOTE:** `PKGBUILD`'s `source` is the rolling `mcmodlist-release.tar.gz`
> asset on the GitHub "latest" release (the version is in the release title,
> not the asset name). `sha256sums` is a placeholder until that asset exists;
> run `updpkgsums` after the first release is published.

## Build locally

```sh
cd packaging/arch
makepkg -si            # release PKGBUILD (needs a real source tarball first)
makepkg -si -f PKGBUILD-git   # git PKGBUILD once the repo URL is real
```

## Test in a clean chroot

```sh
# extra-x86_64-build (from devtools) or makechrootpkg
extra-x86_64-build -r /var/cache/pacman/pkg/
```

## Cutting a release

Releases are published by GitHub Actions (`.github/workflows/ci.yml`):

1. Push a tag: `git tag vX.Y.Z && git push origin vX.Y.Z` (or run the
   workflow manually via *Actions → ci → Run workflow*).
2. After the `go` and `qt` jobs pass, the `release` job archives the source
   as `mcmodlist-release.tar.gz` (fixed top-level dir `mcmodlist/`) and
   creates a GitHub release titled `mcmodlist X.Y.Z` with that asset.
3. `PKGBUILD`'s `source` always points at
   `releases/latest/download/mcmodlist-release.tar.gz`, so it never needs a
   per-version URL. Bump `pkgver` in `PKGBUILD` to match, then run
   `updpkgsums` and regenerate `.SRCINFO`:

   ```sh
   makepkg --printsrcinfo > .SRCINFO
   ```

## Push to the AUR

1. `git clone ssh://aur@aur.archlinux.org/mcmodlist.git` (and
   `mcmodlist-git.git` for the -git package).
2. Copy `PKGBUILD`, `.SRCINFO`, and `LICENSE` into the repo.
3. `makepkg --printsrcinfo > .SRCINFO` inside the AUR clone, commit, push.

## Lint

```sh
namcap PKGBUILD
namcap mcmodlist-*.pkg.tar.zst mcmodlist-gui-*.pkg.tar.zst
```
