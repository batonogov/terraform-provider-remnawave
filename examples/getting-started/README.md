# Getting started

This example creates one Remnawave user with a 10 GiB monthly traffic limit.
It keeps the API token outside Terraform configuration and can be removed cleanly
after verification.

## Prerequisites

- A running Remnawave panel in the supported 2.7.x–3.2.x range
- Terraform 1.x
- An API token created in the Remnawave panel

## Run

```sh
export REMNAWAVE_ENDPOINT="https://panel.example.com"
export REMNAWAVE_API_TOKEN="..."

terraform init
terraform plan
terraform apply
```

Check the created object without printing the sensitive subscription URL:

```sh
terraform state show remnawave_user.quickstart
```

When finished, remove the example user:

```sh
terraform destroy
```

## Adopt an existing panel

Define each existing object in HCL first, then use the import command documented
on its Registry resource page. Import identifiers are resource-specific: for
example, Remnawave 3.x users use a numeric ID, while Remnawave 2.x users use a
UUID. Run `terraform plan` after every import and reconcile configuration before
applying changes.

The root [README](../../README.md#importing-existing-resources) contains import
examples, and every supported resource page includes its exact import syntax.

## State safety

Terraform state can contain sensitive provider and resource values. Store state
in a protected backend, restrict access to snapshots and backups, and never
commit credentials or secret-bearing `.tfvars` files.
