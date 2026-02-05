# PostgreSQL RLS Benchmark

## Running Benchmarks

```bash
make bench
```

## Results

Took ~23 minutes

```
goos: linux
goarch: amd64
pkg: github.com/flanksource/duty/bench
cpu: Intel(R) Core(TM) i9-14900K
BenchmarkMain/Sample-10000/catalog_changes/Without_RLS-32                   6649           1620014 ns/op
BenchmarkMain/Sample-10000/catalog_changes/With_RLS-32                      3399           3547842 ns/op
BenchmarkMain/Sample-10000/config_changes/Without_RLS-32                    7155         1628757 ns/op
BenchmarkMain/Sample-10000/config_changes/With_RLS-32                       3273           3569723 ns/op
BenchmarkMain/Sample-10000/config_detail/Without_RLS-32                     9540         1220023 ns/op
BenchmarkMain/Sample-10000/config_detail/With_RLS-32                        4063           2900145 ns/op
BenchmarkMain/Sample-10000/config_names/Without_RLS-32                      1663           7124439 ns/op
BenchmarkMain/Sample-10000/config_names/With_RLS-32                         4093           2914901 ns/op
BenchmarkMain/Sample-10000/config_summary/Without_RLS-32                    687            17482952 ns/op
BenchmarkMain/Sample-10000/config_summary/With_RLS-32                       5929           1908932 ns/op
BenchmarkMain/Sample-10000/configs/Without_RLS-32                           5686           2078947 ns/op
BenchmarkMain/Sample-10000/configs/With_RLS-32                              4071           2906051 ns/op
BenchmarkMain/Sample-10000/analysis_types/Without_RLS-32                    9468           1266709 ns/op
BenchmarkMain/Sample-10000/analysis_types/With_RLS-32                       9462           1251793 ns/op
BenchmarkMain/Sample-10000/analyzer_types/Without_RLS-32                    9788           1199055 ns/op
BenchmarkMain/Sample-10000/analyzer_types/With_RLS-32                       9859           1214105 ns/op
BenchmarkMain/Sample-10000/change_types/Without_RLS-32                      7276           1601141 ns/op
BenchmarkMain/Sample-10000/change_types/With_RLS-32                         7311           1619463 ns/op
BenchmarkMain/Sample-10000/config_classes/Without_RLS-32                   10000           1045498 ns/op
BenchmarkMain/Sample-10000/config_classes/With_RLS-32                       4072           2904409 ns/op
BenchmarkMain/Sample-10000/config_types/Without_RLS-32                      9136           1223087 ns/op
BenchmarkMain/Sample-10000/config_types/With_RLS-32                         4093           2904356 ns/op
BenchmarkMain/Sample-25000/catalog_changes/Without_RLS-32                   3142           3764216 ns/op
BenchmarkMain/Sample-25000/catalog_changes/With_RLS-32                      1412           8327931 ns/op
BenchmarkMain/Sample-25000/config_changes/Without_RLS-32                    3159           3766311 ns/op
BenchmarkMain/Sample-25000/config_changes/With_RLS-32                       1400           8388122 ns/op
BenchmarkMain/Sample-25000/config_detail/Without_RLS-32                     3972           2967181 ns/op
BenchmarkMain/Sample-25000/config_detail/With_RLS-32                        1696           7008540 ns/op
BenchmarkMain/Sample-25000/config_names/Without_RLS-32                       709          17180999 ns/op
BenchmarkMain/Sample-25000/config_names/With_RLS-32                         1700           6991508 ns/op
BenchmarkMain/Sample-25000/config_summary/Without_RLS-32                     264          45680070 ns/op
BenchmarkMain/Sample-25000/config_summary/With_RLS-32                       2690           4537575 ns/op
BenchmarkMain/Sample-25000/configs/Without_RLS-32                           2382           5012024 ns/op
BenchmarkMain/Sample-25000/configs/With_RLS-32                              1699           6932345 ns/op
BenchmarkMain/Sample-25000/analysis_types/Without_RLS-32                    3981           2994821 ns/op
BenchmarkMain/Sample-25000/analysis_types/With_RLS-32                       4100           2963487 ns/op
BenchmarkMain/Sample-25000/analyzer_types/Without_RLS-32                    4102           2872676 ns/op
BenchmarkMain/Sample-25000/analyzer_types/With_RLS-32                       4158           2865456 ns/op
BenchmarkMain/Sample-25000/change_types/Without_RLS-32                      3058           3953717 ns/op
BenchmarkMain/Sample-25000/change_types/With_RLS-32                         3061           3909598 ns/op
BenchmarkMain/Sample-25000/config_classes/Without_RLS-32                    4725           2566520 ns/op
BenchmarkMain/Sample-25000/config_classes/With_RLS-32                       1682           6972777 ns/op
BenchmarkMain/Sample-25000/config_types/Without_RLS-32                      3924           2963325 ns/op
BenchmarkMain/Sample-25000/config_types/With_RLS-32                         1708           7065202 ns/op
BenchmarkMain/Sample-50000/catalog_changes/Without_RLS-32                   1478           8000063 ns/op
BenchmarkMain/Sample-50000/catalog_changes/With_RLS-32                       674          18089184 ns/op
BenchmarkMain/Sample-50000/config_changes/Without_RLS-32                    1530           8402061 ns/op
BenchmarkMain/Sample-50000/config_changes/With_RLS-32                        669          17876571 ns/op
BenchmarkMain/Sample-50000/config_detail/Without_RLS-32                     2131           5745608 ns/op
BenchmarkMain/Sample-50000/config_detail/With_RLS-32                         866          13684545 ns/op
BenchmarkMain/Sample-50000/config_names/Without_RLS-32                       366          32851181 ns/op
BenchmarkMain/Sample-50000/config_names/With_RLS-32                          868          13829836 ns/op
BenchmarkMain/Sample-50000/config_summary/Without_RLS-32                     124          94852697 ns/op
BenchmarkMain/Sample-50000/config_summary/With_RLS-32                       1329           8875333 ns/op
BenchmarkMain/Sample-50000/configs/Without_RLS-32                           1190           9905524 ns/op
BenchmarkMain/Sample-50000/configs/With_RLS-32                               870          13808263 ns/op
BenchmarkMain/Sample-50000/analysis_types/Without_RLS-32                    1952           6000208 ns/op
BenchmarkMain/Sample-50000/analysis_types/With_RLS-32                       2071           5783666 ns/op
BenchmarkMain/Sample-50000/analyzer_types/Without_RLS-32                    2136           5642676 ns/op
BenchmarkMain/Sample-50000/analyzer_types/With_RLS-32                       2142           5594963 ns/op
BenchmarkMain/Sample-50000/change_types/Without_RLS-32                      1528           7845923 ns/op
BenchmarkMain/Sample-50000/change_types/With_RLS-32                         1544           7853844 ns/op
BenchmarkMain/Sample-50000/config_classes/Without_RLS-32                    2418           4906577 ns/op
BenchmarkMain/Sample-50000/config_classes/With_RLS-32                        871          13753221 ns/op
BenchmarkMain/Sample-50000/config_types/Without_RLS-32                      2029           5746793 ns/op
BenchmarkMain/Sample-50000/config_types/With_RLS-32                          867          13773012 ns/op
BenchmarkMain/Sample-100000/catalog_changes/Without_RLS-32                   640          16708249 ns/op
BenchmarkMain/Sample-100000/catalog_changes/With_RLS-32                      309          37262982 ns/op
BenchmarkMain/Sample-100000/config_changes/Without_RLS-32                    637          16886999 ns/op
BenchmarkMain/Sample-100000/config_changes/With_RLS-32                       319          36727055 ns/op
BenchmarkMain/Sample-100000/config_detail/Without_RLS-32                    1014          11893316 ns/op
BenchmarkMain/Sample-100000/config_detail/With_RLS-32                        426          28133108 ns/op
BenchmarkMain/Sample-100000/config_names/Without_RLS-32                      169          71342338 ns/op
BenchmarkMain/Sample-100000/config_names/With_RLS-32                         428          28080877 ns/op
BenchmarkMain/Sample-100000/config_summary/Without_RLS-32                     67         170440224 ns/op
BenchmarkMain/Sample-100000/config_summary/With_RLS-32                       652          18252059 ns/op
BenchmarkMain/Sample-100000/configs/Without_RLS-32                           573          20778969 ns/op
BenchmarkMain/Sample-100000/configs/With_RLS-32                              423          28208192 ns/op
BenchmarkMain/Sample-100000/analysis_types/Without_RLS-32                    974          12216983 ns/op
BenchmarkMain/Sample-100000/analysis_types/With_RLS-32                      1047          11827838 ns/op
BenchmarkMain/Sample-100000/analyzer_types/Without_RLS-32                   1076          11213405 ns/op
BenchmarkMain/Sample-100000/analyzer_types/With_RLS-32                      1057          11392111 ns/op
BenchmarkMain/Sample-100000/change_types/Without_RLS-32                      639          17009622 ns/op
BenchmarkMain/Sample-100000/change_types/With_RLS-32                         627          16996126 ns/op
BenchmarkMain/Sample-100000/config_classes/Without_RLS-32                   1158           9950993 ns/op
BenchmarkMain/Sample-100000/config_classes/With_RLS-32                       433          27732173 ns/op
BenchmarkMain/Sample-100000/config_types/Without_RLS-32                      990          11939862 ns/op
BenchmarkMain/Sample-100000/config_types/With_RLS-32                         434          27360176 ns/op
```

## Improvements

Changed from

```sql
CASE WHEN is_rls_disabled() THEN TRUE
```

to

```sql
CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
```

causes parsing of `request.jwt.claims` to be cached and offers significant improvements.

Reference: https://github.com/PostgREST/postgrest-docs/issues/609

```
> benchstat bench/old.txt bench/new.txt

goos: linux
goarch: amd64
pkg: github.com/flanksource/duty/bench
cpu: Intel(R) Core(TM) i9-14900K
                                                  │ bench/old.txt  │             bench/new.txt             │
                                                  │     sec/op     │    sec/op     vs base                 │
Main/Sample-10000/catalog_changes/Without_RLS-32      1.618m ± ∞ ¹   1.620m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-10000/catalog_changes/With_RLS-32        25.696m ± ∞ ¹   3.548m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-10000/config_changes/Without_RLS-32       1.663m ± ∞ ¹   1.629m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-10000/config_changes/With_RLS-32         25.400m ± ∞ ¹   3.570m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-10000/config_detail/Without_RLS-32        1.259m ± ∞ ¹   1.220m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-10000/config_detail/With_RLS-32          10.922m ± ∞ ¹   2.900m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-10000/config_names/Without_RLS-32         7.167m ± ∞ ¹   7.124m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-10000/config_names/With_RLS-32           11.294m ± ∞ ¹   2.915m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-10000/config_summary/Without_RLS-32       17.31m ± ∞ ¹   17.48m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-10000/config_summary/With_RLS-32         88.562m ± ∞ ¹   1.909m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-10000/configs/Without_RLS-32              2.071m ± ∞ ¹   2.079m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-10000/configs/With_RLS-32                10.702m ± ∞ ¹   2.906m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-10000/analysis_types/Without_RLS-32       1.268m ± ∞ ¹   1.267m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-10000/analysis_types/With_RLS-32          1.249m ± ∞ ¹   1.252m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-10000/analyzer_types/Without_RLS-32       1.187m ± ∞ ¹   1.199m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-10000/analyzer_types/With_RLS-32          1.208m ± ∞ ¹   1.214m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-10000/change_types/Without_RLS-32         1.603m ± ∞ ¹   1.601m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-10000/change_types/With_RLS-32            1.618m ± ∞ ¹   1.619m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-10000/config_classes/Without_RLS-32       1.054m ± ∞ ¹   1.045m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-10000/config_classes/With_RLS-32         10.602m ± ∞ ¹   2.904m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-10000/config_types/Without_RLS-32         1.221m ± ∞ ¹   1.223m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-10000/config_types/With_RLS-32           10.688m ± ∞ ¹   2.904m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-25000/catalog_changes/Without_RLS-32      3.777m ± ∞ ¹   3.764m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-25000/catalog_changes/With_RLS-32        59.728m ± ∞ ¹   8.328m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-25000/config_changes/Without_RLS-32       3.796m ± ∞ ¹   3.766m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-25000/config_changes/With_RLS-32         59.201m ± ∞ ¹   8.388m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-25000/config_detail/Without_RLS-32        2.825m ± ∞ ¹   2.967m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-25000/config_detail/With_RLS-32          25.408m ± ∞ ¹   7.009m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-25000/config_names/Without_RLS-32         16.34m ± ∞ ¹   17.18m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-25000/config_names/With_RLS-32           26.697m ± ∞ ¹   6.992m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-25000/config_summary/Without_RLS-32       43.75m ± ∞ ¹   45.68m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-25000/config_summary/With_RLS-32        242.723m ± ∞ ¹   4.538m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-25000/configs/Without_RLS-32              4.886m ± ∞ ¹   5.012m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-25000/configs/With_RLS-32                25.306m ± ∞ ¹   6.932m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-25000/analysis_types/Without_RLS-32       2.932m ± ∞ ¹   2.995m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-25000/analysis_types/With_RLS-32          2.936m ± ∞ ¹   2.963m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-25000/analyzer_types/Without_RLS-32       2.777m ± ∞ ¹   2.873m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-25000/analyzer_types/With_RLS-32          2.812m ± ∞ ¹   2.865m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-25000/change_types/Without_RLS-32         3.865m ± ∞ ¹   3.954m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-25000/change_types/With_RLS-32            3.806m ± ∞ ¹   3.910m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-25000/config_classes/Without_RLS-32       2.435m ± ∞ ¹   2.567m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-25000/config_classes/With_RLS-32         25.212m ± ∞ ¹   6.973m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-25000/config_types/Without_RLS-32         2.862m ± ∞ ¹   2.963m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-25000/config_types/With_RLS-32           25.352m ± ∞ ¹   7.065m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-50000/catalog_changes/Without_RLS-32      7.545m ± ∞ ¹   8.000m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-50000/catalog_changes/With_RLS-32        117.27m ± ∞ ¹   18.09m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-50000/config_changes/Without_RLS-32       7.552m ± ∞ ¹   8.402m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-50000/config_changes/With_RLS-32         117.77m ± ∞ ¹   17.88m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-50000/config_detail/Without_RLS-32        5.593m ± ∞ ¹   5.746m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-50000/config_detail/With_RLS-32           49.42m ± ∞ ¹   13.68m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-50000/config_names/Without_RLS-32         31.77m ± ∞ ¹   32.85m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-50000/config_names/With_RLS-32            52.55m ± ∞ ¹   13.83m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-50000/config_summary/Without_RLS-32       90.89m ± ∞ ¹   94.85m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-50000/config_summary/With_RLS-32        473.003m ± ∞ ¹   8.875m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-50000/configs/Without_RLS-32              9.465m ± ∞ ¹   9.906m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-50000/configs/With_RLS-32                 49.84m ± ∞ ¹   13.81m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-50000/analysis_types/Without_RLS-32       5.801m ± ∞ ¹   6.000m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-50000/analysis_types/With_RLS-32          5.712m ± ∞ ¹   5.784m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-50000/analyzer_types/Without_RLS-32       5.442m ± ∞ ¹   5.643m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-50000/analyzer_types/With_RLS-32          5.515m ± ∞ ¹   5.595m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-50000/change_types/Without_RLS-32         7.553m ± ∞ ¹   7.846m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-50000/change_types/With_RLS-32            7.634m ± ∞ ¹   7.854m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-50000/config_classes/Without_RLS-32       4.780m ± ∞ ¹   4.907m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-50000/config_classes/With_RLS-32          49.65m ± ∞ ¹   13.75m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-50000/config_types/Without_RLS-32         5.559m ± ∞ ¹   5.747m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-50000/config_types/With_RLS-32            49.52m ± ∞ ¹   13.77m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-100000/catalog_changes/Without_RLS-32     15.79m ± ∞ ¹   16.71m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-100000/catalog_changes/With_RLS-32       236.59m ± ∞ ¹   37.26m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-100000/config_changes/Without_RLS-32      15.86m ± ∞ ¹   16.89m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-100000/config_changes/With_RLS-32        237.73m ± ∞ ¹   36.73m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-100000/config_detail/Without_RLS-32       11.28m ± ∞ ¹   11.89m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-100000/config_detail/With_RLS-32          98.80m ± ∞ ¹   28.13m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-100000/config_names/Without_RLS-32        68.28m ± ∞ ¹   71.34m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-100000/config_names/With_RLS-32          105.50m ± ∞ ¹   28.08m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-100000/config_summary/Without_RLS-32      169.6m ± ∞ ¹   170.4m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-100000/config_summary/With_RLS-32        984.13m ± ∞ ¹   18.25m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-100000/configs/Without_RLS-32             19.59m ± ∞ ¹   20.78m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-100000/configs/With_RLS-32                99.83m ± ∞ ¹   28.21m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-100000/analysis_types/Without_RLS-32      11.43m ± ∞ ¹   12.22m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-100000/analysis_types/With_RLS-32         11.45m ± ∞ ¹   11.83m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-100000/analyzer_types/Without_RLS-32      10.68m ± ∞ ¹   11.21m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-100000/analyzer_types/With_RLS-32         10.85m ± ∞ ¹   11.39m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-100000/change_types/Without_RLS-32        15.86m ± ∞ ¹   17.01m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-100000/change_types/With_RLS-32           16.16m ± ∞ ¹   17.00m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-100000/config_classes/Without_RLS-32      9.487m ± ∞ ¹   9.951m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-100000/config_classes/With_RLS-32         98.95m ± ∞ ¹   27.73m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-100000/config_types/Without_RLS-32        11.28m ± ∞ ¹   11.94m ± ∞ ¹        ~ (p=1.000 n=1) ²
Main/Sample-100000/config_types/With_RLS-32           99.58m ± ∞ ¹   27.36m ± ∞ ¹        ~ (p=1.000 n=1) ²
geomean                                               13.45m         7.163m        -46.76%
¹ need >= 6 samples for confidence interval at level 0.95
² need >= 4 samples to detect a difference at alpha level 0.05

```

- Overall geometric mean shows a 46.76% performance improvement (from 13.45m to 7.163m)
- Specific Notable Improvements: `config_summary` with RLS
- Operations showing the most dramatic RLS improvements:
  - catalog_changes: ~6-7x speedup with RLS
  - config_changes: ~6-7x speedup with RLS
  - config_summary: ~50-54x speedup with RLS
  - config_detail, config_names, configs: ~3-4x speedup with RLS



## With Clauses appended

Testing out if using the rls payload as WHERE clauses improves performance

```
pkg: github.com/flanksource/duty/bench
cpu: Intel(R) Core(TM) i9-14900K
BenchmarkMain/Sample-10000/catalog_changes/With_RLS-With-Clause-32                  2026           5783302 ns/op
BenchmarkMain/Sample-10000/catalog_changes/With_RLS-32                              2899           3849648 ns/op
BenchmarkMain/Sample-10000/catalog_changes/Without_RLS-32                           7126           1632845 ns/op
BenchmarkMain/Sample-10000/config_detail/With_RLS-With-Clause-32                    4602           2538843 ns/op
BenchmarkMain/Sample-10000/config_detail/With_RLS-32                                4842           2398737 ns/op
BenchmarkMain/Sample-10000/config_detail/Without_RLS-32                             9142           1239478 ns/op
BenchmarkMain/Sample-10000/configs/With_RLS-With-Clause-32                          4412           2632808 ns/op
BenchmarkMain/Sample-10000/configs/With_RLS-32                                      4032           2483985 ns/op
BenchmarkMain/Sample-10000/configs/Without_RLS-32                                   5593           2128625 ns/op
BenchmarkMain/Sample-25000/catalog_changes/With_RLS-With-Clause-32                   852          13981123 ns/op
BenchmarkMain/Sample-25000/catalog_changes/With_RLS-32                              1249           9420119 ns/op
BenchmarkMain/Sample-25000/catalog_changes/Without_RLS-32                           3096           3879005 ns/op
BenchmarkMain/Sample-25000/config_detail/With_RLS-With-Clause-32                    2006           5863158 ns/op
BenchmarkMain/Sample-25000/config_detail/With_RLS-32                                2037           5856141 ns/op
BenchmarkMain/Sample-25000/config_detail/Without_RLS-32                             3697           2977967 ns/op
BenchmarkMain/Sample-25000/configs/With_RLS-With-Clause-32                          1962           6106637 ns/op
BenchmarkMain/Sample-25000/configs/With_RLS-32                                      1992           6027257 ns/op
BenchmarkMain/Sample-25000/configs/Without_RLS-32                                   2283           5059330 ns/op
BenchmarkMain/Sample-50000/catalog_changes/With_RLS-With-Clause-32                   408          29456885 ns/op
BenchmarkMain/Sample-50000/catalog_changes/With_RLS-32                               594          19630053 ns/op
BenchmarkMain/Sample-50000/catalog_changes/Without_RLS-32                           1465           8190458 ns/op
BenchmarkMain/Sample-50000/config_detail/With_RLS-With-Clause-32                     994          12086096 ns/op
BenchmarkMain/Sample-50000/config_detail/With_RLS-32                                1033          11413711 ns/op
BenchmarkMain/Sample-50000/config_detail/Without_RLS-32                             2044           5827890 ns/op
BenchmarkMain/Sample-50000/configs/With_RLS-With-Clause-32                           973          12323323 ns/op
BenchmarkMain/Sample-50000/configs/With_RLS-32                                      1023          11772035 ns/op
BenchmarkMain/Sample-50000/configs/Without_RLS-32                                   1150          10265080 ns/op
BenchmarkMain/Sample-100000/catalog_changes/With_RLS-With-Clause-32                  213          55548502 ns/op
BenchmarkMain/Sample-100000/catalog_changes/With_RLS-32                              288          39636005 ns/op
BenchmarkMain/Sample-100000/catalog_changes/Without_RLS-32                           626          17153132 ns/op
BenchmarkMain/Sample-100000/config_detail/With_RLS-With-Clause-32                    487          24382695 ns/op
BenchmarkMain/Sample-100000/config_detail/With_RLS-32                                514          23039022 ns/op
BenchmarkMain/Sample-100000/config_detail/Without_RLS-32                             972          12149169 ns/op
BenchmarkMain/Sample-100000/configs/With_RLS-With-Clause-32                          482          24730473 ns/op
BenchmarkMain/Sample-100000/configs/With_RLS-32                                      495          23903118 ns/op
BenchmarkMain/Sample-100000/configs/Without_RLS-32                                   555          21545467 ns/op
```