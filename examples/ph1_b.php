<?php
echo "--- PHP CPU Benchmark (Single Thread) ---\n";

$start = microtime(true);
$limit = 50000000;
$total = 0;

for ($i = 0; $i < $limit; $i++) {
    if ($i < 2) {
        continue;
    }
    $isPrime = true;
    for ($j = 2; $j * $j <= $i; $j++) {
        if ($i % $j == 0) {
            $isPrime = false;
            break;
        }
    }
    if ($isPrime) {
        $total += $i;
    }
}

$elapsed = microtime(true) - $start;
echo "Result: {$total}\n";
echo "Time: {$elapsed} sec\n";
?>
