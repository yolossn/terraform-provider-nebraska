data "nebraska_channel" "demo" {
  application_id = "io.kinvolk.demo"
  arch           = "amd64"
  name           = "Demo channel name"
}


output "channel_id" {
  value = data.nebraska_channel.demo.id
}
