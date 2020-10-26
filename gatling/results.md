```
RESULTS
================================================================================
---- Global Information --------------------------------------------------------
> request count                                     101000 (OK=101000 KO=0     )
> min response time                                      0 (OK=0      KO=-     )
> max response time                                     56 (OK=56     KO=-     )
> mean response time                                     8 (OK=8      KO=-     )
> std deviation                                          4 (OK=4      KO=-     )
> response time 50th percentile                          8 (OK=8      KO=-     )
> response time 75th percentile                         10 (OK=10     KO=-     )
> response time 95th percentile                         15 (OK=15     KO=-     )
> response time 99th percentile                         21 (OK=21     KO=-     )
> mean requests/sec                                3884.615 (OK=3884.615 KO=-  )
---- Response Time Distribution ------------------------------------------------
> t < 800 ms                                        101000 (100%)
> 800 ms < t < 1200 ms                                   0 (  0%)
> t > 1200 ms                                            0 (  0%)
> failed                                                 0 (  0%)
================================================================================
```

Note : These load tests were run into Docker container for both Souin and gatling. Then these container were running on MacOS i5 (3.1 GHz) + 8Go RAM. Some benchmarks will be available in future depending each platform
