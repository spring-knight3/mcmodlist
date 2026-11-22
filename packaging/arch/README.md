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

1. Tag the release (`git tag vX.Y.Z`) and push.
2. In `packaging/arch/PKGBUILD`: bump `pkgver`, fix `url=`/`source=` to the
   real repository, then run `updpkgsums`.
3. Regenerate `.SRCINFO`:

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
