resource "nebraska_application" "demo_app" {
  product_id  = "io.kinvolk.demo"
  name        = "Demo app"
  description = "demo app is used for demo purpose"
}

resource "nebraska_channel" "demo_channel" {
  arch           = "amd64"
  name           = "Demo channel name"
  application_id = nebraska_application.demo_app.id
}
