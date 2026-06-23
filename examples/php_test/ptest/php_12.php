<?php

echo "----------------------------------------\n";
echo "    PHP Prime Benchmark\n";
echo "----------------------------------------\n";

$start = microtime(true);
$limit = 500000;

$totalPrimes = 0;

for ($i = 2; $i <= $limit; $i++) {
    $isPrime = true;

    for ($j = 2; $j * $j <= $i; $j++) {
        if ($i % $j == 0) {
            $isPrime = false;
            break;
        }
    }

    if ($isPrime) {
        $totalPrimes++;
    }
}

$elapsed = microtime(true) - $start;

echo "Target Limit    : $limit\n";
echo "Primes Found    : $totalPrimes\n";
echo "Execution Time  : $elapsed seconds\n";
echo "----------------------------------------\n";