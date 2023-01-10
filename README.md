# mackerel-plugin-dns-synthetic


## Usage

```
% ./mackerel-plugin-dns-synthetic --prefix dns -H ns1 -H ns2 -Q example.com. -E 192.0.0.1 --timeout 5s
dns8-synthetic.service.available        100     1672992548
dns8-synthetic.status.error     0       1672992548
dns8-synthetic.status.success   2       1672992548
dns8-synthetic.rtt.max  28      1672992548
dns8-synthetic.rtt.mean 20.500000       1672992548
```

## Help

```
Usage:
  mackerel-plugin-dns-synthetic [OPTIONS]

Application Options:
  -v, --version            Show version
      --prefix=            Metric key prefix (default: dnsdist)
      --protocol=[tcp|udp]
  -p, --port=              Port number (default: 53)
  -H, --hostname=          DNS server hostnames (default: 127.0.0.1)
  -Q, --question=          Question hostname (default: example.com.)
  -E, --expect=            Expect string in result
      --timeout=           Timeout (default: 5s)

Help Options:
  -h, --help               Show this help message
```
