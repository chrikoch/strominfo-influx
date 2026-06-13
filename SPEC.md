# scope

strominfo-influx should do the following:

* pull frequency data from the enegry charts api (see https://api.energy-charts.info/#/power/frequency_frequency_get) and write it into influx
* pull day ahead price data from the enegry charts api (see https://api.energy-charts.info/#/prices/day_ahead_price_price_get) and write it into influx


## frequency

* There is no need to pull old data twice, as the measured frequency will not change afterwards.
* There will no data be availble for future timestamps. Trying to fetch them might even trigger errors in the API
* frequency is fetched for the region "DE-Freiburg".

## price

* Day ahead prices will be available for the next day at about ~13:00h (Berlin timezone). Once fetched, they will not change.


## overall

* We should be kind to the provider of the API, so not pulling data too often. Nevertheless the data pushed to influx is used to visualize intra-day graphs in grafana, so having the most current data every ~15 minutes would be great.
* There is no need to save a "last fetched timestamp" between restarts of strominfo-influx. We simply start fresh every time and fetch for the current timestamp or perhaps the current day, but not we will not need to pull all old data.
* The energy charts API expects start and end parameters as unix timestamps. So they are UTC. The local timezone setting of the host might and will differ.
* Prices are fetched for the zone "DE-LU".