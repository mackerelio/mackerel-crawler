```
mackerel-crawler --provider aws --key xxx
```

- `--provider` metric provider like aws

## AWS provider

- `--aws-key-id` api key for fetching metric from provider
- `--aws-secret-key` api secret key for fetching metric from provider
- `--aws-region` AWS region

- ELB, RDS are posted as independent hosts
- `--service` target service to which fetcher posts metrics


## Fastly provider

- `--key` api key for fetching metric from provider
- `--service` target service to which fetcher posts metrics
- metrics are posted as service metrics
