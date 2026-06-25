resource "laravelforge_team_invites" "example" {
  email        = "example-email"
  organization = "my-org"
  role_id      = 1
  team         = "example-team"
}
