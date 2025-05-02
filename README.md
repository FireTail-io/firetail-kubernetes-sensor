# Firetail Kubernetes Sensor

Quickstart:

```bash
git clone git@github.com:FireTail-io/firetail-kubernetes-sensor.git
docker build . -t firetail-kubernetes-sensor && docker run -p 8080:8080 firetail-kubernetes-sensor
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
