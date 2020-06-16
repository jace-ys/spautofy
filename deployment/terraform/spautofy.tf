provider "random" {
  version = "~> 2.2"
}

variable "spotify_client_id" {
  type        = string
  description = "Spotify client ID."
}

variable "spotify_client_secret" {
  type        = string
  description = "Spotify client secret."
}

variable "sendgrid_api_key" {
  type        = string
  description = "API key for accessing the SendGrid API."
}

variable "sendgrid_sender_name" {
  type        = string
  description = "Name to use when sending mail via SendGrid."
}

variable "sendgrid_sender_email" {
  type        = string
  description = "Email to use when sending mail via SendGrid."
}

variable "sendgrid_template_id" {
  type        = string
  description = "Template ID to use when sending mail via SendGrid."
}

resource "random_password" "session_store_key" {
  length = 24
}

resource "heroku_app_config_association" "spautofy" {
  app_id = heroku_app.main.id

  vars = {
    BASE_URL = heroku_app.main.web_url
  }

  sensitive_vars = {
    SESSION_STORE_KEY     = random_password.session_store_key.result
    SPOTIFY_CLIENT_ID     = var.spotify_client_id
    SPOTIFY_CLIENT_SECRET = var.spotify_client_secret
    SENDGRID_API_KEY      = var.sendgrid_api_key
    SENDGRID_SENDER_NAME  = var.sendgrid_sender_name
    SENDGRID_SENDER_EMAIL = var.sendgrid_sender_email
    SENDGRID_TEMPLATE_ID  = var.sendgrid_template_id
  }
}
