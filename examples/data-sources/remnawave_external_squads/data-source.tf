data "remnawave_external_squads" "all" {}

output "all_external_squads" {
  value = data.remnawave_external_squads.all.external_squads
}
