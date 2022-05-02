data "nebraska_group" "demo" {
  application_id = "io.kinvolk.demo"
  name           = "demo group"
}


output "group_id" {
  value = data.nebraska_group.demo.id
}
