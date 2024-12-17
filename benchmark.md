# PostgreSQL RLS Benchmark

## Running Benchmarks

query duration to fetch 10k, 25k, 50k and 100k config items in random are recorded.

```bash
# With RLS
go run main.go

# Without RLS
go run main.go --disable-rls
```

## Results

### Without RLS

```json
[
  {
    "config_count": 10000,
    "duration": 766282108,
    "rls_enabled": false
  },
  {
    "config_count": 25000,
    "duration": 1964247115,
    "rls_enabled": false
  },
  {
    "config_count": 50000,
    "duration": 4738917435,
    "rls_enabled": false
  },
  {
    "config_count": 100000,
    "duration": 7663285526,
    "rls_enabled": false
  }
]
```

| Configuration Count | Duration (s) |
| ------------------- | ------------ |
| 10,000              | 766.3        |
| 25,000              | 1.964        |
| 50,000              | 4.738        |
| 100,000             | 7.663        |

### With RLS

```json
[
  {
    "config_count": 10000,
    "duration": 897432497,
    "rls_enabled": true
  },
  {
    "config_count": 25000,
    "duration": 1794868472,
    "rls_enabled": true
  },
  {
    "config_count": 50000,
    "duration": 4383678988,
    "rls_enabled": true
  },
  {
    "config_count": 100000,
    "duration": 7650471155,
    "rls_enabled": true
  }
]
```

| Configuration Count | Duration (s) |
| ------------------- | ------------ |
| 10,000              | 897.4        |
| 25,000              | 1.794        |
| 50,000              | 4.383        |
| 100,000             | 7.650        |
