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

## Download

Set the release once, then download its assets into an empty directory:

```sh
repo=batonogov/terraform-provider-remnawave
tag=v0.6.2
version=${tag#v}
mkdir "remnawave-$tag"
cd "remnawave-$tag"
gh release download "$tag" --repo "$repo"
```

Use the version you intend to install; do not assume the example is current.

## Verify checksums and the GPG signature

Import the project release public key from a trusted, independent source before
running this step. Then verify the detached signature and every file covered by
the checksum manifest:

```sh
checksums="terraform-provider-remnawave_${version}_SHA256SUMS"
gpg --verify "${checksums}.sig" "$checksums"
shasum -a 256 -c "$checksums"
```

On Linux, `sha256sum -c "$checksums"` is equivalent. The Registry-compatible
checksum file contains exactly the provider ZIPs and the Registry manifest;
SBOM integrity is covered by the provenance bundle below. Stop if either
command fails.

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
