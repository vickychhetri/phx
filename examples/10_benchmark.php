<?php
// -------------------------------------------------------
// Example 10: Performance Benchmark
// -------------------------------------------------------

echo "========================================\n";
echo "    PHX Performance Benchmark\n";
echo "========================================\n";

// --- Test 1: Arithmetic loop ---
$start = microtime(true);
$sum   = 0;
for ($i = 1; $i <= 1000000; $i++) {
    $sum += $i;
}
$t1 = microtime(true) - $start;
echo "Arithmetic sum(1..1M) = " . $sum . "\n";
echo "Time: " . $t1 . "s\n\n";

// --- Test 2: Prime sieve ---
$start = microtime(true);
$limit = 50000;
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
        $count++;
    }
}
$t2 = microtime(true) - $start;
echo "Primes up to " . $limit . " = " . $count . "\n";
echo "Time: " . $t2 . "s\n\n";

// --- Test 3: Fibonacci (recursive) ---
function fib($n) {
    if ($n <= 1) { return $n; }
    return fib($n - 1) + fib($n - 2);
}
$start = microtime(true);
$f     = fib(30);
$t3    = microtime(true) - $start;
echo "fib(30) = " . $f . "\n";
echo "Time: " . $t3 . "s\n\n";

// --- Test 4: String concatenation ---
$start = microtime(true);
$s     = "";
for ($i = 0; $i < 10000; $i++) {
    $s = $s . "x";
}
$t4 = microtime(true) - $start;
echo "String concat (10K) length = " . strlen($s) . "\n";
echo "Time: " . $t4 . "s\n\n";

echo "========================================\n";
echo "Total time: " . ($t1 + $t2 + $t3 + $t4) . "s\n";
echo "========================================\n";
?>
