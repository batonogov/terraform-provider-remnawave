# Repository and release security

The files under `.github/repository-settings/` are the source of truth for
repository controls that cannot be represented by workflow YAML. They protect
the default branch, `v*` release tags, Actions execution, and the `release`
deployment environment.

## Rollout order

Apply the settings only after the PRs adding the `Vulnerability Scan`,
`Release Gate Tests`, and `Release Artifact Tests` jobs have reached `main`.
Requiring a check before its workflow exists blocks every pull request.

1. Add a reviewer other than the person who normally merges a release PR. Get
   the reviewer's numeric ID with `gh api users/LOGIN --jq .id`.
2. Apply and immediately audit the committed policy:

   ```sh
   RELEASE_REVIEWER_ID=123456 \
     ./scripts/configure-repository-security.sh --apply
   ./scripts/configure-repository-security.sh --check
   ```

3. Add `RELEASE_GPG_PRIVATE_KEY` and `RELEASE_GPG_PASSPHRASE` as secrets on the
   `release` Environment. Test a non-production signing run, then remove the old
   repository-level `GPG_PRIVATE_KEY` and `GPG_PASSPHRASE` secrets. GitHub does
   not expose existing secret values, so they must be supplied again.
4. After the draft-first release flow is in place, enable **Release
   immutability** under **Settings → General → Releases**. This setting applies
   only to future releases and prevents published assets and their tags from
   being changed.

The release job cannot read Environment secrets until the required reviewer
approves the deployment. Self-review is disabled. All other jobs retain
read-only default permissions; write permissions are granted only on the jobs
that update the release PR or publish artifacts.

## Emergency bypass

The committed rulesets define no bypass actors. If an incident makes a bypass
unavoidable:

1. record the reason, affected refs, approver, and time window in a private
   security incident;
2. add one named actor with `pull_request`-only bypass where possible;
3. make the minimum change through an auditable pull request;
4. remove the bypass immediately and rerun the audit script;
5. rotate release credentials if the workflow or Environment boundary may have
   been exposed.

Never move or reuse a published version tag. Ship a new version instead.
