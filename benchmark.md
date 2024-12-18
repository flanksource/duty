# PostgreSQL RLS Benchmark

## Running Benchmarks

query duration to fetch 10k, 25k, 50k and 100k config items in random are recorded.

```bash
make bench
```

## Results

### RLS Enabled

```
goos: linux
goarch: amd64
pkg: github.com/flanksource/duty/cmd/bench
cpu: Intel(R) Core(TM) i9-14900K
BenchmarkFetchConfigNames
BenchmarkFetchConfigNames/FetchConfigNames-10000
BenchmarkFetchConfigNames/FetchConfigNames-10000-32                    3         395821592 ns/op        69434000 B/op    1220122 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-10000-32                    4         401464958 ns/op        69498028 B/op    1220152 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-10000-32                    2         610787302 ns/op        69517464 B/op    1220164 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-10000-32                    3         365766762 ns/op        69585378 B/op    1220169 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-10000-32                    2         502813822 ns/op        69441808 B/op    1220138 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-25000
BenchmarkFetchConfigNames/FetchConfigNames-25000-32                    2         879523447 ns/op        173763224 B/op   3050378 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-25000-32                    2         844931444 ns/op        173763744 B/op   3050380 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-25000-32                    2         884904475 ns/op        173557068 B/op   3050331 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-25000-32                    1        1091559988 ns/op        173808128 B/op   3050415 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-25000-32                    2        1149870354 ns/op        173953248 B/op   3050447 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-50000
BenchmarkFetchConfigNames/FetchConfigNames-50000-32                    1        2337716384 ns/op        347523920 B/op   6100993 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-50000-32                    1        1835866645 ns/op        347604528 B/op   6100800 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-50000-32                    1        1411237052 ns/op        347790272 B/op   6100834 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-50000-32                    1        2094802520 ns/op        347519776 B/op   6100690 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-50000-32                    1        1752965788 ns/op        347640184 B/op   6100790 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-100000
BenchmarkFetchConfigNames/FetchConfigNames-100000-32                   1        4829827750 ns/op        694315112 B/op  12201544 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-100000-32                   1        3506304646 ns/op        694776408 B/op  12201497 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-100000-32                   1        3215697217 ns/op        694583768 B/op  12201404 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-100000-32                   1        3020625572 ns/op        694173112 B/op  12201316 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-100000-32                   1        3334046796 ns/op        695182504 B/op  12201532 allocs/op
```

### RLS Disabled

```
goos: linux
goarch: amd64
pkg: github.com/flanksource/duty/cmd/bench
cpu: Intel(R) Core(TM) i9-14900K
BenchmarkFetchConfigNames
BenchmarkFetchConfigNames/FetchConfigNames-10000
BenchmarkFetchConfigNames/FetchConfigNames-10000-32                    3         387588345 ns/op        69448485 B/op    1220129 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-10000-32                    5         334666455 ns/op        69658137 B/op    1220197 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-10000-32                    3         439962778 ns/op        69523461 B/op    1220163 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-10000-32                    3         386959435 ns/op        69548330 B/op    1220167 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-10000-32                    2         549263874 ns/op        69672360 B/op    1220201 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-25000
BenchmarkFetchConfigNames/FetchConfigNames-25000-32                    1        1627014841 ns/op        173691096 B/op   3050608 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-25000-32                    1        1420053298 ns/op        173956744 B/op   3050435 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-25000-32                    1        1029687054 ns/op        174033896 B/op   3050474 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-25000-32                    2        1066024193 ns/op        173951604 B/op   3050434 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-25000-32                    2        1022169710 ns/op        174084652 B/op   3050482 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-50000
BenchmarkFetchConfigNames/FetchConfigNames-50000-32                    1        2110668725 ns/op        347526712 B/op   6101023 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-50000-32                    1        2142099541 ns/op        347604336 B/op   6100805 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-50000-32                    1        2860712580 ns/op        347414032 B/op   6100730 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-50000-32                    1        1842813888 ns/op        347640816 B/op   6100804 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-50000-32                    1        1832628431 ns/op        347339400 B/op   6100713 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-100000
BenchmarkFetchConfigNames/FetchConfigNames-100000-32                   1        3996763325 ns/op        695260608 B/op  12201845 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-100000-32                   1        3437644582 ns/op        695674376 B/op  12201700 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-100000-32                   1        3981860250 ns/op        695412200 B/op  12201643 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-100000-32                   1        3892324760 ns/op        694736720 B/op  12201473 allocs/op
BenchmarkFetchConfigNames/FetchConfigNames-100000-32                   1        3231683488 ns/op        694961528 B/op  12201516 allocs/op
```
