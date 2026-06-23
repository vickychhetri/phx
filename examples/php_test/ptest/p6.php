<?php
echo "----------------------------------------\n";
echo "    PHX Execution Performance Benchmark \n";
echo "----------------------------------------\n";

$start = microtime(true);

$limit = 500000;
$count = 0;

for ($i = 2; $i <= $limit; $i++) {
    $isPrime = true;
    for ($j = 2; $j * $j <= $i; $j++) {
        if ($i % $j == 0) {
            $isPrime = false;
            break;
        }
    }
    if ($isPrime) {
        $count = $count + 1;
    }
}

$end = microtime(true);
$elapsed = $end - $start;

echo "Target Limit    : " . $limit . "\n";
echo "Primes Found    : " . $count . "\n";
echo "Execution Time  : " . $elapsed . " seconds\n";
echo "----------------------------------------\n";
?>