![Build status](https://github.com/function61/function53/workflows/Build/badge.svg)
[![Download](https://img.shields.io/github/downloads/function61/function53/total.svg?style=for-the-badge)](https://github.com/function61/function53/releases)
[![Download](https://img.shields.io/docker/pulls/fn61/function53.svg?style=for-the-badge)](https://hub.docker.com/r/fn61/function53/)

What
----

A DNS server for your LAN that blocks ads/malware and encrypts your DNS traffic.

Designed to work on Raspberry Pi (much like Pi-hole), but works elsewhere as well.

![](docs/metrics.png)


Why
---

I wanted these features:

- Ad blocking
- Encrypted DNS (DoH or DoT)
- Operational metrics
- Metrics should include query latencies
- Clean install with single binary

And here's the alternatives' feature matrix:

| Project        | Ad blocking | Encrypted DNS | Metrics | Latency distribution metrics | Clean install | Reasonable programming language |
|----------------|-------------|---------------|---------|-----------------|---------------|---------------|
| function53     | ✓           | ✓             | ✓       | ✓               | ✓             | ✓ (Go)        |
| dnscrypt-proxy | ✓           | ✓             | [Not coming](https://github.com/jedisct1/dnscrypt-pro✓y/issues/337) |   | ✓ | ✓ (Go) |
| coredns        |             | ✓             | ✓       | ✓               | ✓             | ✓ (Go) |
| pihole         | ✓           | Manual config | [Plugin](https://github.com/eko/pihole-exporter)  |              | No | No (PHP) |
| AdGuard        | ✓           | ✓             | [Not yet](https://github.com/AdguardTeam/AdGuardHome/issues/516) | No |   | ✓ (Go) |

Definitions:

| Term          | Definition                    |
|---------------|-------------------------------|
| Metrics       | Prometheus-compatible metrics |
| Clean install | Minimal changes to the system (Pi-hole needs so many dependencies, and even the Docker image for Pi-hole looks too complicated.) |
| Reasonable programming language | Uses memory safe, typed programming language |

I also had [reliability problems with dnscrypt-proxy](https://github.com/coredns/coredns/issues/2267#issuecomment-450131975).


How to install
--------------

This assumes you're using Raspberry Pi. The URL is different for amd64.

```
$ mkdir ~/function53 && cd ~/function53/
$ VERSION_TO_DOWNLOAD="..." # find this from Bintray. Looks like: 20180828_1449_b9d7759cf80f0b4a
$ sudo curl --location --fail --output function53 "https://dl.bintray.com/function61/dl/function53/$VERSION_TO_DOWNLOAD/function53_linux-arm" && sudo chmod +x function53
$ ./function53 write-default-config
$ cat config.json

... inspect the configuration to see if it suits you

$ ./function53 write-systemd-unit-file
Wrote unit file to /etc/systemd/system/function53.service
Run to enable on boot & to start now:
        $ systemctl enable function53
        $ systemctl start function53
        $ systemctl status function53
```

NOTE: You may need `$ sudo` for some of those commands.

After starting function53, check with `$ dig` or `$ nslookup` (nslookup works the same on
Windows if you like it better) that the name resolution works:

```
$ nslookup joonas.fi <ip of your DNS server>
$ dig joonas.fi @<ip of your DNS server>
```

To test ad blocking, lookup `adtech.de`.


Links
-----

- https://github.com/jedisct1/dnscrypt-proxy/wiki/Public-blacklists
- https://dnscrypt.info/public-servers/


How to build & develop
----------------------

[How to build & develop](https://github.com/function61/turbobob/blob/master/docs/external-how-to-build-and-dev.md)
(with Turbo Bob, our build tool). It's easy and simple!
