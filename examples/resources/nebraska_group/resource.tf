resource "nebraska_application" "demo_app" {
  product_id  = "io.kinvolk.demo"
  name        = "Demo app"
  description = "demo app is used for demo purpose"
}

resource "nebraska_group" "demo_group" {
  name           = "demo group"
  application_id = nebraska_application.demo_app.id
}
