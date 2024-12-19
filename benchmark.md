# PostgreSQL RLS Benchmark

## Running Benchmarks

query duration to fetch 10k, 25k, 50k and 100k config items in random are recorded.

```bash
make bench
```

## Data

```
| Tags                      | Count   |
| ------------------------- | ------- |
| `{"cluster": "aws"}`      | 38,556  |
| `{"cluster": "azure"}`    | 25,704  |
| `{"cluster": "demo"}`     | 27,846  |
| `{"cluster": "gcp"}`      | 29,274  |
| `{"cluster": "homelab"}`  | 33,558  |
| `{"region": "eu-west-1"}` | 24,990  |
| `{"region": "eu-west-2"}` | 22,134  |
| `{"region": "us-east-1"}` | 23,562  |
| `{"region": "us-east-2"}` | 26,418  |
| **Total**                 | 252,042 |

```

## Query Plans

### Without RLS

```sql
EXPLAIN (ANALYZE, BUFFERS) SELECT id FROM config_names WHERE id = 'fe1a8a69-8eeb-4771-bc9b-8d1c4a6b6b52'
+---------------------------------------------------------------------------------------------------------------------------------------------+
| QUERY PLAN                                                                                                                                  |
|---------------------------------------------------------------------------------------------------------------------------------------------|
| Subquery Scan on config_names  (cost=8.45..8.46 rows=1 width=16) (actual time=0.020..0.020 rows=1 loops=1)                                  |
|   Buffers: shared hit=4                                                                                                                     |
|   ->  Sort  (cost=8.45..8.45 rows=1 width=90) (actual time=0.020..0.020 rows=1 loops=1)                                                     |
|         Sort Key: config_items.name                                                                                                         |
|         Sort Method: quicksort  Memory: 25kB                                                                                                |
|         Buffers: shared hit=4                                                                                                               |
|         ->  Index Scan using config_items_pkey on config_items  (cost=0.42..8.44 rows=1 width=90) (actual time=0.011..0.012 rows=1 loops=1) |
|               Index Cond: (id = 'fe1a8a69-8eeb-4771-bc9b-8d1c4a6b6b52'::uuid)                                                               |
|               Buffers: shared hit=4                                                                                                         |
| Planning Time: 0.078 ms                                                                                                                     |
| Execution Time: 0.029 ms                                                                                                                    |
+---------------------------------------------------------------------------------------------------------------------------------------------+
```

### With RLS

```sql
EXPLAIN (ANALYZE, BUFFERS) SELECT id FROM config_names WHERE id = 'fe1a8a69-8eeb-4771-bc9b-8d1c4a6b6b52'
+---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------+
| QUERY PLAN                                                                                                                                                                                                                                                                                                                      |
|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Subquery Scan on config_names  (cost=13.03..13.04 rows=1 width=16) (actual time=0.019..0.019 rows=0 loops=1)                                                                                                                                                                                                                    |
|   Buffers: shared hit=4                                                                                                                                                                                                                                                                                                         |
|   ->  Sort  (cost=13.03..13.03 rows=1 width=90) (actual time=0.019..0.019 rows=0 loops=1)                                                                                                                                                                                                                                       |
|         Sort Key: config_items.name                                                                                                                                                                                                                                                                                             |
|         Sort Method: quicksort  Memory: 25kB                                                                                                                                                                                                                                                                                    |
|         Buffers: shared hit=4                                                                                                                                                                                                                                                                                                   |
|         InitPlan 1 (returns $0)                                                                                                                                                                                                                                                                                                 |
|           ->  Result  (cost=0.00..3.28 rows=100 width=16) (actual time=0.001..0.002 rows=0 loops=1)                                                                                                                                                                                                                             |
|                 ->  ProjectSet  (cost=0.00..0.53 rows=100 width=32) (actual time=0.001..0.001 rows=0 loops=1)                                                                                                                                                                                                                   |
|                       ->  Result  (cost=0.00..0.01 rows=1 width=0) (actual time=0.000..0.000 rows=1 loops=1)                                                                                                                                                                                                                    |
|         ->  Index Scan using config_items_pkey on config_items  (cost=0.42..9.74 rows=1 width=90) (actual time=0.018..0.018 rows=0 loops=1)                                                                                                                                                                                     |
|               Index Cond: (id = 'fe1a8a69-8eeb-4771-bc9b-8d1c4a6b6b52'::uuid)                                                                                                                                                                                                                                                   |
|               Filter: CASE WHEN ((current_setting('request.jwt.claims'::text, true) IS NULL) OR (current_setting('request.jwt.claims'::text, true) = ''::text) OR (((current_setting('request.jwt.claims'::text, true))::jsonb ->> 'disable_rls'::text) IS NOT NULL)) THEN true ELSE ((agent_id = ANY ($0)) OR (SubPlan 2)) END |
|               Rows Removed by Filter: 1                                                                                                                                                                                                                                                                                         |
|               Buffers: shared hit=4                                                                                                                                                                                                                                                                                             |
|               SubPlan 2                                                                                                                                                                                                                                                                                                         |
|                 ->  Function Scan on jsonb_array_elements allowed_tags  (cost=0.02..1.27 rows=1 width=0) (actual time=0.005..0.005 rows=0 loops=1)                                                                                                                                                                              |
|                       Filter: (config_items.tags @> value)                                                                                                                                                                                                                                                                      |
|                       Rows Removed by Filter: 1                                                                                                                                                                                                                                                                                 |
| Planning Time: 0.093 ms                                                                                                                                                                                                                                                                                                         |
| Execution Time: 0.030 ms                                                                                                                                                                                                                                                                                                        |
+---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------+
```

## Results

```
using tags map[cluster:azure]

goos: linux
goarch: amd64
pkg: github.com/flanksource/duty/cmd/bench
cpu: Intel(R) Core(TM) i9-14900K
BenchmarkFetchConfigNames
BenchmarkFetchConfigNames/WithoutRLS
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-10000
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-10000-32                 2         245365822 ns/op        69215156 B/op    1220075 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-10000-32                 2         261091446 ns/op        69216036 B/op    1220081 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-10000-32                 2         255445556 ns/op        69252972 B/op    1220085 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-10000-32                 2         244023555 ns/op        69326344 B/op    1220088 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-10000-32                 2         278308284 ns/op        69196596 B/op    1220071 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-25000
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-25000-32                 2         920743781 ns/op        173309360 B/op   3050246 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-25000-32                 2         629957600 ns/op        173252256 B/op   3050226 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-25000-32                 2         894046993 ns/op        173063576 B/op   3050167 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-25000-32                 2         964531232 ns/op        173198600 B/op   3050237 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-25000-32                 2         601064311 ns/op        172953196 B/op   3050162 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-50000
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-50000-32                 2        1435745944 ns/op        346282188 B/op   6100424 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-50000-32                 2        1696153102 ns/op        346188368 B/op   6100396 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-50000-32                 2        1556478618 ns/op        346131304 B/op   6100376 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-50000-32                 2        1420193476 ns/op        346320304 B/op   6100438 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-50000-32                 2        1912309306 ns/op        346392728 B/op   6100429 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-100000
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-100000-32                2        2816319512 ns/op        692391640 B/op  12200764 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-100000-32                2        3293933604 ns/op        692527740 B/op  12200848 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-100000-32                2        3154926514 ns/op        692848684 B/op  12200920 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-100000-32                2        3384120826 ns/op        692992728 B/op  12200966 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-100000-32                2        3058929510 ns/op        692565596 B/op  12200853 allocs/op

BenchmarkFetchConfigNames/WithRLS
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-10000
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-10000-32                    2         288283234 ns/op        67774056 B/op    1166482 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-10000-32                    2         328436766 ns/op        67717524 B/op    1166464 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-10000-32                    2         335607736 ns/op        67716080 B/op    1166448 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-10000-32                    2         266702067 ns/op        67660940 B/op    1166446 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-10000-32                    2         280315924 ns/op        67661092 B/op    1166448 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-25000
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-25000-32                    2         862582546 ns/op        169106208 B/op   2915848 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-25000-32                    2         969516421 ns/op        169086624 B/op   2915832 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-25000-32                    2         773097155 ns/op        169200744 B/op   2915879 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-25000-32                    2         814463218 ns/op        169296440 B/op   2915922 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-25000-32                    2         876850648 ns/op        169103320 B/op   2915816 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-50000
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-50000-32                    2        1442559590 ns/op        338496376 B/op   5836311 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-50000-32                    2        1586663000 ns/op        338442048 B/op   5836318 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-50000-32                    2        1726732958 ns/op        338384768 B/op   5836292 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-50000-32                    2        1666323980 ns/op        338425196 B/op   5836331 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-50000-32                    2        1521879976 ns/op        338460348 B/op   5836317 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-100000
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-100000-32                   2        3263852292 ns/op        676876696 B/op  11673257 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-100000-32                   2        3378996070 ns/op        676785336 B/op  11673260 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-100000-32                   2        3275695985 ns/op        677270488 B/op  11673355 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-100000-32                   2        3343083346 ns/op        677272988 B/op  11673386 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-100000-32                   2        3445708884 ns/op        677118412 B/op  11673295 allocs/op
```

```
using tags map[cluster:homelab]

goos: linux
goarch: amd64
pkg: github.com/flanksource/duty/cmd/bench
cpu: Intel(R) Core(TM) i9-14900K
BenchmarkFetchConfigNames
BenchmarkFetchConfigNames/WithoutRLS
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-10000
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-10000-32                 2         465555848 ns/op        69159256 B/op    1220064 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-10000-32                 2         235456148 ns/op        69196448 B/op    1220070 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-10000-32                 2         409922865 ns/op        69290992 B/op    1220099 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-10000-32                 2         329581220 ns/op        69327520 B/op    1220098 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-10000-32                 2         434752796 ns/op        69403920 B/op    1220131 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-25000
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-25000-32                 2         825174844 ns/op        173007692 B/op   3050158 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-25000-32                 2         752771215 ns/op        173027592 B/op   3050174 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-25000-32                 2         517989336 ns/op        173064296 B/op   3050175 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-25000-32                 2        1268070917 ns/op        173142536 B/op   3050224 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-25000-32                 2         704331920 ns/op        172953000 B/op   3050160 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-50000
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-50000-32                 2        1454420899 ns/op        346450852 B/op   6100464 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-50000-32                 2        1331816381 ns/op        346209704 B/op   6100429 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-50000-32                 2        1442663195 ns/op        346129136 B/op   6100355 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-50000-32                 2        1675077925 ns/op        346110932 B/op   6100355 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-50000-32                 2        1617782833 ns/op        346169348 B/op   6100392 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-100000
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-100000-32                2        2878183626 ns/op        692447520 B/op  12200773 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-100000-32                2        3008883030 ns/op        692638444 B/op  12200850 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-100000-32                2        2906402836 ns/op        692338656 B/op  12200783 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-100000-32                2        3166923536 ns/op        692657472 B/op  12200859 allocs/op
BenchmarkFetchConfigNames/WithoutRLS/FetchConfigNames-100000-32                2        2667958748 ns/op        692465760 B/op  12200775 allocs/op

BenchmarkFetchConfigNames/WithRLS
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-10000
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-10000-32                    2         349448531 ns/op        67568372 B/op    1166643 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-10000-32                    2         287953222 ns/op        67686688 B/op    1166682 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-10000-32                    2         278567598 ns/op        67724816 B/op    1166698 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-10000-32                    2         283438036 ns/op        67568164 B/op    1166640 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-10000-32                    2         279482918 ns/op        67742992 B/op    1166696 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-25000
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-25000-32                    2         763302121 ns/op        169233784 B/op   2922709 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-25000-32                    2         730491592 ns/op        169458052 B/op   2922756 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-25000-32                    2        1059253540 ns/op        169309152 B/op   2922733 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-25000-32                    2         712281172 ns/op        169327384 B/op   2922731 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-25000-32                    2         885699035 ns/op        169570400 B/op   2922784 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-50000
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-50000-32                    2        1603425474 ns/op        338190904 B/op   5830467 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-50000-32                    2        1600269898 ns/op        338020768 B/op   5830413 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-50000-32                    2        1450531206 ns/op        338172736 B/op   5830468 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-50000-32                    2        1412282198 ns/op        338019324 B/op   5830394 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-50000-32                    2        1510709330 ns/op        338151748 B/op   5830439 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-100000
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-100000-32                   2        2905927532 ns/op        676843604 B/op  11672867 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-100000-32                   2        3343012506 ns/op        676635836 B/op  11672804 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-100000-32                   2        3416915146 ns/op        676465608 B/op  11672743 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-100000-32                   2        3293499296 ns/op        676822272 B/op  11672837 allocs/op
BenchmarkFetchConfigNames/WithRLS/FetchConfigNames-100000-32                   2        2972546026 ns/op        676840692 B/op  11672841 allocs/op
```