# COVID Dashboard for Grafana
This repository contains dashboard for Grafana that shows actual COVID-19 data for selected Country.

Data comes from [Corona Data Scraper](https://coronadatascraper.com/) under Public Domain License.

## Caution
The data is missing some values therefore some countries might not show all values.

## Contents
The repository contains of several components:
* agent source that fetches data every 15 minutes from Corona Data Scraper and inserts it into InfluxDB database
* Dashboard Panel JSON for Grafana.
* docker-compose.yaml defining services for running it locally

## Running it locally
To run it locally you need to have:
* Docker
* docker-compose

To run all needed components run:
```shell script
docker-compose up -d
```

Run grafana instance under http://localhost:3000 and login using default initial credentials. Configure datasource as follows:
```
Datasource: InfluxDB
Address: http://influxdb:8086
Database: covid
```
Import Panel JSON and use previously added datasource.

