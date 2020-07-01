[![spautofy-badge]][spautofy-workflow]

[spautofy-badge]: https://github.com/jace-ys/spautofy/workflows/spautofy/badge.svg
[spautofy-workflow]: https://github.com/jace-ys/spautofy/actions?query=workflow%3Aspautofy

# Spautofy

Automated creation of Spotify playlists based on your recent top tracks. Check it out at https://spautofy.herokuapp.com.

## Prerequisites 

- go
- [go-bindata](https://github.com/kevinburke/go-bindata)
- docker, docker-compose

## Usage

1. Create a .env file in the root directory containing the following environment variables:

```shell
SPOTIFY_CLIENT_ID=
SPOTIFY_CLIENT_SECRET=
SENDGRID_API_KEY=
SENDGRID_SENDER_NAME=
SENDGRID_SENDER_EMAIL=
SENDGRID_TEMPLATE_ID=
```

2. Start auxiliary containers:

```
make dependencies
```

3. Compile web assets:

```
make assets
```

4. Run the Spautofy server:

```
make
```

5. Access it at http://localhost:8080

You can also start all services directly via a single command:

```
docker-compose up
```

### Metrics

The following endpoints are available on the metrics server at http://localhost:9090:

```shell
/metrics # view HTTP server metrics
/health  # view liveness and readiness
/crons   # view all currently scheduled crons
```

## Deployment

Spautofy is automatically deployed to Heroku on push to master, after Continuous Integration checks have all passed. Any pre-deployment tasks, such as database migrations, are ran as part of the deployment process using Heroku's release phase.

Heroku resources are provisioned via Terraform located in [deployment/terraform](https://github.com/jace-ys/spautofy/tree/master/deployment/terraform).

[cron-job.org](https://cron-job.org/en) is used to ping the Spautofy server every 15 mins to keep the web process alive round the clock.
