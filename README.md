# 2-Phase Character Generator

[![Go Reference](https://pkg.go.dev/badge/github.com/tommie/chargen2p.svg)](https://pkg.go.dev/github.com/tommie/chargen2p)

This project is a random data generator that can be used to test
network performance (more specifically: throughput).

The original chargen protocol defined in [RFC
864](https://datatracker.ietf.org/doc/html/rfc864) simply says the
server will listen for connections and immediately start streaming
random data. The UDP version sends one datagram per datagram received.

The server in this project first waits on TCP data and then sends back
the same amount. This is a bit safer against write-amplification
exploits, and also means we can measure both upload and download,
separately. There is no UDP version.

## Server Usage

Install `chargen2pd` on a server, e.g. using the Systemd files in
`etc/`. A `Dockerfile` is also provided. Since I consider this a
spiritual successor of the original chargen, the default is to bind to
TCP port 19.

```shell
$ go mod download
$ go run ./cmd/chargen2pd -listen-addr tcp:localhost:1919
2006/01/02 15:16:17 Listening for connections on 127.0.0.1:1919...
```

## Client Usage

The `testthroughput` program sends and receives some data against a
server.

```shell
$ go mod download
$ go run ./cmd/testthroughput -addr localhost:1919
Uploaded 670040064 bytes in 1.584983235s: 3381.941 Mbps
Downloaded 670040064 bytes in 1.630415568s: 3287.702 Mbps
```

## Protocol

See the code in [`conn.go`](https://github.com/tommie/chargen2p/blob/main/conn.go).

It surprised me to read that speedtest.net just [repeats the same
short string over and
over](https://gist.github.com/sdstrowes/411fca9d900a846a704f68547941eb97). This
is trivial for an ISP to compress and still claim plausible
deniability.

## Security

This is an unauthenticated and unencrypted service. It sets
concurrency limits, but if you run this on a public server, you may
create significant egress traffic. It is not an echo server, to avoid
being used as a relay with a spoofed source address.

The code itself is simple Go and should be fine.

## License

This project is licensed under the [MIT License](LICENSE).
