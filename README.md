# Firetail Kubernetes Sensor

POC for a FireTail Kubernetes Sensor.



## Quickstart

Clone the repo, then use the `run` target in [the provided makefile](./Makefile):

```bash
git clone git@github.com:FireTail-io/firetail-kubernetes-sensor.git
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

```bash
2025/05/02 13:27:15 🧐 Listening for packets on port 8080...
2025/05/02 13:27:16 😭 Failed to parse packet no payload found in TCP layer
2025/05/02 13:27:16 😭 Failed to parse packet no payload found in TCP layer
2025/05/02 13:27:16 ✅ Received packet from 172.17.0.1:42812 to 172.17.0.3:8080 with payload:
----------START----------
GET /world HTTP/1.1
Host: localhost:8080
User-Agent: curl/8.7.1
Accept: */*


-----------END-----------
2025/05/02 13:27:16 😭 Failed to parse packet no payload found in TCP layer
2025/05/02 13:27:16 😭 Failed to parse packet no payload found in TCP layer
2025/05/02 13:27:16 😭 Failed to parse packet no payload found in TCP layer
```



## Publishing to ECS

The `publish` target in [the provided makefile](./Makefile) will login to ECS, tag the image and push it:

```bash
make publish
```
