[![Build Status](https://img.shields.io/travis/function61/function53.svg?style=for-the-badge)](https://travis-ci.org/function61/function53)
[![Download](https://img.shields.io/bintray/v/function61/function53/main.svg?style=for-the-badge&label=Download)](https://bintray.com/function61/function53/main/_latestVersion#files)

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

Metrics for dnscrypt-proxy [may not ever be coming](https://github.com/jedisct1/dnscrypt-proxy/issues/337).

I define "clean install" as minimal changes to the system. Pi-hole needs so many dependencies,
and even the Docker image for Pi-hole looks too complicated.

I also had [reliability problems with dnscrypt-proxy](https://github.com/coredns/coredns/issues/2267#issuecomment-450131975).


How to install
--------------

This assumes you're using Raspberry Pi. The URL is different for amd64.

```
$ mkdir function53 && cd function53/
$ VERSION_TO_DOWNLOAD="..." # find this from Bintray. Looks like: 20180828_1449_b9d7759cf80f0b4a
$ sudo curl --location --fail --output function53 "https://dl.bintray.com/function61/function53/$VERSION_TO_DOWNLOAD/function53_linux-arm" && sudo chmod +x function53
$ ./function53 write-systemd-unit-file
Wrote unit file to /etc/systemd/system/function53.service
Run to enable on boot & to start now:
        $ systemctl enable function53
        $ systemctl start function53
        $ systemctl status function53
```


Links
-----

- https://github.com/jedisct1/dnscrypt-proxy/wiki/Public-blacklists
- https://dnscrypt.info/public-servers/


How to build & develop
----------------------

[How to build & develop](https://github.com/function61/turbobob/blob/master/docs/external-how-to-build-and-dev.md)
(with Turbo Bob, our build tool). It's easy and simple!
