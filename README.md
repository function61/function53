[![Build Status](https://img.shields.io/travis/function61/function53.svg?style=for-the-badge)](https://travis-ci.org/function61/function53)
[![Download](https://img.shields.io/docker/pulls/fn61/function53.svg?style=for-the-badge)](https://hub.docker.com/r/fn61/function53/)

What
----

A DNS server for your LAN that blocks ads/malware and encrypts your DNS traffic.

Designed to work on Raspberry Pi (much like Pi-hole), but works elsewhere as well.


Why
---

I wanted these features:

- Ad blocking
- Encrypted DNS (DoH or DoT)
- Operational metrics
- Metrics should include query latencies
- Clean install with single binary

And here's the alternatives' feature matrix:

| Project        | Ad blocking | Encrypted DNS | Metrics | Latency metrics | Clean install | Not using PHP |
|----------------|-------------|---------------|---------|-----------------|---------------|---------------|
| function53     | x           | x             | x       | x               | x             | x             |
| dnscrypt-proxy | x           | x             |         |                 | x             | x             |
| coredns        |             | x             | x       | x               | x             | x             |
| pihole         | x           | Manual config | x       |                 |               |               |

Metrics for [dnscrypt-proxy](https://github.com/jedisct1/dnscrypt-proxy/issues/337) are probably never coming.

I define clean install as minimal changes to the system. Pi-hole needs so many dependencies,
and even the Docker image for Pi-hole looks too complicated.

I also had [reliability problems with dnscrypt-proxy](https://github.com/coredns/coredns/issues/2267#issuecomment-450131975).


Links
-----

- https://github.com/jedisct1/dnscrypt-proxy/wiki/Public-blacklists
- https://dnscrypt.info/public-servers/
