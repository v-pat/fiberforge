# FiberForge Release Checklist

Use this checklist for every official release of FiberForge.

---

## 1. Pre-Release Validation

Run all test suites and static analysis tools:

```bash
# 1. Go tests, vet, and race detection
go test ./...
go vet ./...
go test -race ./...

# 2. Go vulnerability scanning
govulncheck ./...

# 3. NPM installer test suite
cd npm/fiberforge
npm test
npm audit
cd ../..

# 4. Clean tree verification
git diff --check
git status --short
```

---

## 2. Version Alignment

Ensure version strings are synchronized:
- `npm/fiberforge/package.json`: `"version": "<X.Y.Z>"`
- Target Git tag: `v<X.Y.Z>`

Ensure no binaries are tracked in the working tree:
```bash
git ls-files | grep -E '(^|/)fiberforge$|npm/fiberforge/bin/fiberforge'
# Expected: empty output
```

---

## 3. NPM Package Build Verification

Verify that the npm tarball builds without binary bloat:

```bash
cd npm/fiberforge
npm pack
# Verify unpacked size (~22 kB) and that no precompiled binaries are included in the archive
rm fiberforge-cli-*.tgz
cd ../..
```

---

## 4. Git Tag & GitHub Release

Trigger the automated GoReleaser workflow via Git tag:

```bash
# 1. Commit and push any release preparation changes
git push origin main

# 2. Create and push the release tag
git tag -a v<X.Y.Z> -m "Release v<X.Y.Z>"
git push origin v<X.Y.Z>
```

The GitHub Actions workflow (`.github/workflows/release.yml`) will:
1. Run GoReleaser
2. Compile binaries for all target OS/architectures (darwin, linux, windows / amd64, arm64)
3. Generate `checksums.txt`
4. Attach archives and checksums to the GitHub release

---

## 5. NPM Publication

Once the GitHub release assets and `checksums.txt` are published:

```bash
cd npm/fiberforge
npm publish --access public
cd ../..
```

---

## 6. Post-Publish Smoke Testing

In a clean shell outside the repository:

```bash
# 1. Test CLI via npx
npx fiberforge-cli@latest --version
npx fiberforge-cli@latest --help

# 2. Test project scaffolding
mkdir -p /tmp/ff-smoke-test
cd /tmp/ff-smoke-test
npx fiberforge-cli@latest init --help
```
