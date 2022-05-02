data "nebraska_package" "demo" {
  application_id = "io.kinvolk.demo"
  version        = "0.0.1"
  arch           = "amd64"
}


output "package_id" {
  value = data.nebraska_package.demo.id
}
