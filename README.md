# Firetail Kubernetes Sensor

FireTail Kubernetes Sensor.

## Deployment

- Create an API & API Key on the FireTail Platform (https://www.firetail.ai/docs/create-an-api-token)[https://www.firetail.ai/docs/create-an-api-token]
- deploy helm chart ```helm install firetail-sensor-helm firetail-sensor/ --set apiKey="example"```

## Environment Variables

| Variable Name                                   | Required? | Example                                                      | Description                                                  |
| ----------------------------------------------- | --------- | ------------------------------------------------------------ | ------------------------------------------------------------ |
| `FIRETAIL_API_TOKEN`                            | ✅         | `PS-02-XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX-XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX` | The API token the sensor will use to report logs to FireTail |
| `BPF_EXPRESSION`                                | ❌         | `tcp and (port 80 or port 443)`                              | The BPF filter used by the sensor. See docs for syntax info: https://www.tcpdump.org/manpages/pcap-filter.7.html |
| `MAX_CONTENT_LENGTH`                            | ❌         | `1048576`                                                    | The sensor will only read request or response bodies if their length is less than `MAX_CONTENT_LENGTH` bytes. |
| `ENABLE_ONLY_LOG_JSON`                          | ❌         | `true`                                                       | Enables only logging requests where the content-type implies the payload should be JSON, or the payload is valid JSON regardless of the content-type. |
| `DISABLE_SERVICE_IP_FILTERING`                  | ❌         | `true`                                                       | Disables polling Kubernetes for the IP addresses of services & subsequently ignoring all requests captured that aren't made to one of those IPs. |
| `FIRETAIL_API_URL`                              | ❌         | `https://api.logging.eu-west-1.prod.firetail.app/logs/bulk`  | The API url the sensor will send logs to. Defaults to the EU region production environment. |
| `FIRETAIL_KUBERNETES_SENSOR_DEV_MODE`           | ❌         | `true`                                                       | Enables debug logging when set to `true`, and reduces the max age of a log in a batch to be sent to FireTail. |
| `FIRETAIL_KUBERNETES_SENSOR_DEV_SERVER_ENABLED` | ❌         | `true`                                                       | Enables a demo web server when set to `true`; useful for sending test requests to. |



## Dev Quickstart

Clone the repo, make a `.env` file with your API token in it, then use the `dev` target in [the provided makefile](./Makefile):

```bash
git clone git@github.com:FireTail-io/firetail-kubernetes-sensor.git
cd firetail-kubernetes-sensor
echo FIRETAIL_API_TOKEN=YOUR_API_TOKEN > .env
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




## Publishing to GHCR

Publishing to GHCR is done via GitHub actions found in [./.github/workflows](./.github/workflows).

You can find the images published here: [github.com/firetail-io/firetail-kubernetes-sensor/pkgs/container/kubernetes-sensor](https://github.com/firetail-io/firetail-kubernetes-sensor/pkgs/container/kubernetes-sensor)
