# Firetail Kubernetes Sensor

Quickstart:

```bash
git clone git@github.com:FireTail-io/firetail-kubernetes-sensor.git
docker build . -t ft-bpf-logger && docker run -p 8080:8080 ft-bpf-logger
```

In another terminal:

```bash
curl localhost:8080
```

You should then see logs:

```bash
2025/05/02 13:27:15 🧐 Listening for packets on port 8080...
2025/05/02 13:27:16 😭 Failed to parse packet no payload found in TCP layer
2025/05/02 13:27:16 😭 Failed to parse packet no payload found in TCP layer
2025/05/02 13:27:16 ✅ Received packet from 172.17.0.1:42812 to 172.17.0.3:8080 with payload:
----------START----------
GET / HTTP/1.1
Host: localhost:8080
User-Agent: curl/8.7.1
Accept: */*


-----------END-----------
2025/05/02 13:27:16 😭 Failed to parse packet no payload found in TCP layer
2025/05/02 13:27:16 😭 Failed to parse packet no payload found in TCP layer
2025/05/02 13:27:16 😭 Failed to parse packet no payload found in TCP layer
```
