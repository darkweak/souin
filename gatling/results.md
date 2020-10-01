```
RESULTS
================================================================================
---- Global Information --------------------------------------------------------
> request count                                     101000 (OK=101000 KO=0     )
> min response time                                      0 (OK=0      KO=-     )
> max response time                                    204 (OK=204    KO=-     )
> mean response time                                    30 (OK=30     KO=-     )
> std deviation                                         17 (OK=17     KO=-     )
> response time 50th percentile                         27 (OK=27     KO=-     )
> response time 75th percentile                         41 (OK=41     KO=-     )
> response time 95th percentile                         60 (OK=60     KO=-     )
> response time 99th percentile                         80 (OK=80     KO=-     )
> mean requests/sec                                2195.652 (OK=2195.652 KO=-  )
---- Response Time Distribution ------------------------------------------------
> t < 800 ms                                        101000 (100%)
> 800 ms < t < 1200 ms                                   0 (  0%)
> t > 1200 ms                                            0 (  0%)
> failed                                                 0 (  0%)
================================================================================
```

Note : These load tests were run into Docker container for both Souin and gatling. Then these container were running on MacOS i5 (3.1 GHz) + 8Go RAM. Some benchmarks will be available in future depending each platform
