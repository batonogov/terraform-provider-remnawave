data "remnawave_shared_lists" "all" {}

output "shared_list_sizes" {
  value = {
    for list in data.remnawave_shared_lists.all.shared_lists :
    list.name => list.items_count
  }
}
