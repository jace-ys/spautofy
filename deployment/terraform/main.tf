terraform {
  required_version = ">= 0.12"
}

provider "heroku" {
  version = "~> 2.4"
}

variable "app" {
  type        = string
  description = "Name of the application."
}

resource "heroku_app" "main" {
  name   = var.app
  region = "eu"
  stack  = "container"
}

resource "heroku_addon" "database" {
  app  = heroku_app.main.name
  plan = "heroku-postgresql:hobby-dev"
}
