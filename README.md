# Firetail Kubernetes Sensor

POC for a FireTail Kubernetes Sensor.



## Environment Variables

| Variable Name                                   | Required? | Example                                                      | Description                                                  |
| ----------------------------------------------- | --------- | ------------------------------------------------------------ | ------------------------------------------------------------ |
| `FIRETAIL_API_TOKEN`                            | ✅         | `PS-02-XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX-XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX` | The API token the sensor will use to report logs to FireTail |
| `BPF_EXPRESSION`                                | ❌         | `tcp and (port 80 or port 443)`                              | The BPF filter used by the sensor. See docs for syntax info: https://www.tcpdump.org/manpages/pcap-filter.7.html |
| `FIRETAIL_API_URL`                              | ❌         | `https://api.logging.eu-west-1.prod.firetail.app/logs/bulk`  | The API url the sensor will send logs to. Defaults to the EU region production environment. |
| `FIRETAIL_KUBERNETES_SENSOR_DEV_MODE`           | ❌         | `true`                                                       | Enables debug logging when set to `true`.                    |
| `FIRETAIL_KUBERNETES_SENSOR_DEV_SERVER_ENABLED` | ❌         | `true`                                                       | Enables a demo web server when set to `true`; useful for sending test requests to. |





## Dev Quickstart

Clone the repo, make a `.env` file with your API token in it, then use the `dev` target in [the provided makefile](./Makefile):

```bash
git clone git@github.com:FireTail-io/firetail-kubernetes-sensor.git
cd firetail-kubernetes-sensor
echo FIRETAIL_API_TOKEN=YOUR_API_TOKEN > .env
cd build_setup
make dev
```

In another terminal:

```bash
curl localhost:8080/world
```

You should receive the following response:

```
Hello, world!
```

After a few seconds, you should see logs appear in the FireTail SaaS platform.



## Publishing to ECS

Authenticate and then use the `publish` target in [the provided makefile](./Makefile) to login to ECS, tag the image and push it:

```bash
ftauth
cd build_setup
make publish VERSION=latest
```
