# Verifying releases

Every provider release publishes the following integrity material:

- one ZIP archive for each platform in `release-targets.json`;
- one `<archive>.spdx.json` SPDX 2.3 SBOM for each ZIP;
- `terraform-provider-remnawave_<version>_SHA256SUMS` and its detached GPG
  signature;
- `terraform-provider-remnawave_<version>_provenance.intoto.jsonl`, containing
  GitHub/Sigstore SLSA provenance for all platform archives and SBOMs.

The Terraform Registry verifies its downloaded package against the release
checksums. The commands below independently connect those same archive bytes to
the release tag and the protected GitHub Actions workflow.

## Select a release

Pass the release tag explicitly instead of copying a version from this
documentation:

```sh
: "${TAG:?Set TAG to the release tag you want to verify}"
repo=batonogov/terraform-provider-remnawave
provider=batonogov/remnawave
tag=$TAG
version=${tag#v}
```

The automated smoke verifier also rejects tags that are not strict SemVer with
a `v` prefix.

## Bootstrap the release key

The expected release-key fingerprint is:

```text
CB77 A603 7F6C A36D 514C  8DC5 B6D2 12FC 24D5 A5B1
```

Do not trust a public key merely because it is attached to the same GitHub
release as the signature. The Terraform Registry exposes the provider signing
key in its download metadata, providing an independent bootstrap source. Fetch
the key, require its long key ID to occur exactly once, and compare the complete
fingerprint before importing it:

```sh
expected_fingerprint=CB77A6037F6CA36D514C8DC5B6D212FC24D5A5B1
expected_key_id=B6D212FC24D5A5B1
metadata=$(mktemp)
release_key=$(mktemp)
trap 'rm -f "$metadata" "$release_key"' EXIT

curl -fsSL \
  "https://registry.terraform.io/v1/providers/$provider/$version/download/linux/amd64" \
  >"$metadata"

jq -er --arg key_id "$expected_key_id" '
  [.signing_keys.gpg_public_keys[]
    | select((.key_id | ascii_upcase) == $key_id)
    | .ascii_armor]
  | if length == 1 then .[0] else error("expected exactly one release key") end
' "$metadata" >"$release_key"

actual_fingerprint=$(
  gpg --batch --show-keys --with-colons "$release_key" |
    awk -F: '$1 == "fpr" {print toupper($10); exit}'
)
test "$actual_fingerprint" = "$expected_fingerprint"
gpg --batch --import "$release_key"
```

Stop if the Registry does not return one matching key or if the complete
fingerprint differs.

## Download

Download the selected release assets into an empty directory:

```sh
mkdir "remnawave-$tag"
cd "remnawave-$tag"
gh release download "$tag" --repo "$repo"
```

## Verify checksums and the GPG signature

Verify the detached signature and every file covered by the checksum manifest:

```sh
checksums="terraform-provider-remnawave_${version}_SHA256SUMS"
manifest="terraform-provider-remnawave_${version}_manifest.json"
gpg --verify "${checksums}.sig" "$checksums"
shasum -a 256 -c "$checksums"
```

On Linux, `sha256sum -c "$checksums"` is equivalent. The Registry-compatible
checksum file must contain only provider ZIP archives and the Registry
manifest. Check that policy before trusting the successful hashes:

```sh
awk '{print $2}' "$checksums" |
  awk -v manifest="$manifest" '
    /\.zip$/ { next }
    $0 == manifest { next }
    { print "unexpected checksum entry: " $0 >"/dev/stderr"; exit 1 }
  '
```

SBOM integrity is covered by the provenance bundle below. Stop if any
signature, policy, or checksum command fails.

## Verify provenance

Resolve the release tag to its commit, then verify a downloaded archive against
the published Sigstore bundle. Pinning the repository, workflow, and source
digest prevents a valid attestation from another workflow or commit from being
accepted:

```sh
archive="terraform-provider-remnawave_${version}_linux_amd64.zip"
bundle="terraform-provider-remnawave_${version}_provenance.intoto.jsonl"
tag_sha=$(gh api "repos/$repo/commits/$tag" --jq .sha)

gh attestation verify "$archive" \
  --bundle "$bundle" \
  --repo "$repo" \
  --signer-workflow "$repo/.github/workflows/release-please.yml" \
  --source-digest "$tag_sha"
```

Run the command for every archive you consume. Verification hashes the local
file, checks that exact digest appears as a SLSA provenance subject, and
validates the Sigstore certificate and transparency evidence. Modifying either
the archive or the bundle makes verification fail.

## Inspect the SBOM

The SBOM name is the archive name plus `.spdx.json`. Confirm that it is valid
SPDX 2.3 and inspect its Go modules, versions, and license declarations:

```sh
sbom="${archive}.spdx.json"
jq -e '
  .spdxVersion == "SPDX-2.3"
  and (.packages | type == "array" and length > 0)
  and (.relationships | type == "array" and length > 0)
' "$sbom"

jq -r '
  .packages[]
  | [.name, .versionInfo, .licenseDeclared, .licenseConcluded]
  | @tsv
' "$sbom"
```

Verify the exact SBOM bytes against the same provenance bundle:

```sh
gh attestation verify "$sbom" \
  --bundle "$bundle" \
  --repo "$repo" \
  --signer-workflow "$repo/.github/workflows/release-please.yml" \
  --source-digest "$tag_sha"
```

## Key rotation and revocation

A different fingerprint is a new trust decision, not an automatic update.
Before rotating the key, publish the new complete fingerprint through the
project's independent trust channel, update this document and `SECURITY.md`,
and identify the first release signed with the new key. Keep the previous
fingerprint documented for historical releases.

If the signing key may be compromised, revoke it, replace the protected release
Environment secrets, and publish a new release signed by the replacement key.
Never modify or re-sign an existing release tag.

## Maintainer smoke check

The smoke verifier performs the key bootstrap, checksum policy, signature,
SBOM, and provenance checks in one command:

```sh
./scripts/smoke-verify-published-release.sh "$TAG"
```

Its offline regression test uses generated assets and command fakes, so it is
safe to run in CI without release credentials or network access:

```sh
./scripts/test-smoke-verify-published-release.sh
```
