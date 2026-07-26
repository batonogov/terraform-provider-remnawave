# Repository and release security

The files under `.github/repository-settings/` are the source of truth for
repository controls that cannot be represented by workflow YAML. They protect
the default branch, `v*` release tags, Actions execution, and the `release`
deployment environment.

## Rollout modes

Apply the settings only after the PRs adding the `Vulnerability Scan`,
`Release Gate Tests`, `Release Artifact Tests`, and
`Release Supply Chain Tests` jobs have reached `main`. Requiring a check before
its workflow exists blocks every pull request.

The strict policy requires a second trusted participant. Get that reviewer's
numeric ID with `gh api users/LOGIN --jq .id`, then apply the policy:

```sh
RELEASE_REVIEWER_ID=123456 \
  ./scripts/configure-repository-security.sh --apply
```

Under **Settings → Environments → release**, disable administrator bypass of
deployment protection rules before running the strict audit.

While there is no second trusted participant, use the explicit solo-maintainer
mode instead:

```sh
./scripts/configure-repository-security.sh --apply-solo
```

Solo mode still requires pull requests, signed commits, successful status
checks, protected release tags, read-only default workflow permissions, and
full-SHA-pinned Actions. It deliberately sets the approving-review count to
zero and does not pretend that an independent release approval exists. Upgrade
to the strict policy as soon as a second trusted reviewer is available.

## Release secrets and final audit

After applying either policy:

1. Add `RELEASE_GPG_PRIVATE_KEY` and `RELEASE_GPG_PASSPHRASE` as secrets on the
   `release` Environment. GitHub does not expose existing secret values, so they
   must be supplied again from the original protected copy.
2. Confirm both Environment secrets are listed, then remove the identically
   named repository-level secrets.
3. Run the matching live audit:

   ```sh
   ./scripts/configure-repository-security.sh --check-solo
   # Or, after an independent reviewer is configured:
   ./scripts/configure-repository-security.sh --check
   ```

The audit verifies branch and tag rulesets, the GitHub Actions App identity for
every required check, Actions permissions, release-Environment policy, release
secret scope, and immutable releases.

The release job cannot read Environment secrets until the required reviewer
approves the deployment in strict mode. Self-review and administrator bypass
are disabled there. In solo mode the Environment scopes credentials to the
release job but cannot provide an independent approval boundary. All other jobs
retain read-only default permissions; write permissions are granted only on the
jobs that update the release PR or publish artifacts.

## Live drift audit

The `Repository Security Audit` workflow runs the same solo-mode check on a
schedule and on demand. Configure `REPOSITORY_SECURITY_AUDIT_TOKEN` as a
repository secret containing a fine-grained, read-only token that can inspect
repository administration settings, Actions settings, Environments, and secret
names. It cannot read secret values or mutate repository settings.

When strict review is enabled, change the workflow command from `--check-solo`
to `--check` in the same pull request that promotes the committed policy.

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
