resource "nebraska_application" "demo_app" {
  product_id  = "io.kinvolk.demo"
  name        = "Demo app"
  description = "demo app is used for demo purpose"
}


resource "nebraska_package" "demo_package" {
  type           = "other"
  version        = "0.0.1"
  url            = "http://kinvolk.io"
  filename       = "flatcar_production_update.gz"
  size           = "465881871"
  hash           = "somerandomhash"
  arch           = "amd64"
  application_id = nebraska_application.demo_app.id
  description    = "demo package"
}
