data "remnawave_internal_squads" "all" {}

output "all_internal_squads" {
  value = data.remnawave_internal_squads.all.internal_squads
}
