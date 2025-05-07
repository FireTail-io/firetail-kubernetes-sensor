# Firetail Kubernetes Sensor

POC for a FireTail Kubernetes Sensor.



## Quickstart

Clone the repo, then use the `run` target in [the provided makefile](./Makefile):

```bash
git clone git@github.com:FireTail-io/firetail-kubernetes-sensor.git
cd firetail-kubernetes-sensor
make run
```

In another terminal:

```bash
curl localhost:8080/world
```

You should receive the following response:

```
Hello, world!
```

And the docker container should have logs similar to the following:

```text
2025/05/07 16:23:11 🔍 Starting local HTTP server...
2025/05/07 16:23:11 🔍 Starting HTTP request streamer...
2025/05/07 16:23:11 🔍 Starting HTTP request & response logger...
2025/05/07 16:23:14 📡 Captured HTTP request & response: 
        Request: GET /world 
        Response: 200 OK 
        Host: localhost:8080 
        Request Body:  
        Response Body: Hello, world! 
        Request Headers: map[Accept:[*/*] User-Agent:[curl/8.7.1]] 
        Response Headers: map[Content-Length:[13] Content-Type:[text/plain; charset=utf-8] Date:[Wed, 07 May 2025 16:23:14 GMT]]
```



## Publishing to ECS

Authenticate and then use the `publish` target in [the provided makefile](./Makefile) to login to ECS, tag the image and push it:

```bash
ftauth
make publish
```
