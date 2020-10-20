```
RESULTS
================================================================================
---- Global Information --------------------------------------------------------
> request count                                     101000 (OK=101000 KO=0     )
> min response time                                      0 (OK=0      KO=-     )
> max response time                                    379 (OK=379    KO=-     )
> mean response time                                    12 (OK=12     KO=-     )
> std deviation                                          9 (OK=9      KO=-     )
> response time 50th percentile                         11 (OK=11     KO=-     )
> response time 75th percentile                         14 (OK=14     KO=-     )
> response time 95th percentile                         23 (OK=23     KO=-     )
> response time 99th percentile                         35 (OK=35     KO=-     )
> mean requests/sec                                3482.759 (OK=3482.759 KO=-  )
---- Response Time Distribution ------------------------------------------------
> t < 800 ms                                        101000 (100%)
> 800 ms < t < 1200 ms                                   0 (  0%)
> t > 1200 ms                                            0 (  0%)
> failed                                                 0 (  0%)
================================================================================
```

Note : These load tests were run into Docker container for both Souin and gatling. Then these container were running on MacOS i5 (3.1 GHz) + 8Go RAM. Some benchmarks will be available in future depending each platform
