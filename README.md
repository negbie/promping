# promping

Uses fping to ping various hosts and exposes the results as Prometheus metrics

```bash
Usage of ./promping:
  -d    Dry mode prints only to console
  -i int
        Ping interval in seconds (default 1)
  -p int
        Expose Prometheus metrics on this port. (default 9876)
  -t string
        Single or comma seperated targets (default "localhost")
```