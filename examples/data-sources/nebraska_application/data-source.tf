data "nebraska_application" "demo" {
  product_id = "io.kinvolk.demo"
}


output "application_id" {
  value = data.nebraska_application.demo.id
}
