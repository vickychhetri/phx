<?php
// -------------------------------------------------
// PHX Channel + Thread Benchmark (prime sieve)
// -------------------------------------------------

echo "----------------------------------------\n";
echo "    PHX Channel-Thread Performance\n";
echo "----------------------------------------\n";

$start   = microtime(true);
$limit   = 500000;
$workers = 4;

// channel() with buffer size = number of workers
// so workers never block on send
$chan = channel($workers);

// Each worker gets a contiguous slice of [2, limit]
$chunkSize = intdiv($limit - 1, $workers) + 1;

// -----------------------------------------------------------------
// Spawn one worker per chunk
// -----------------------------------------------------------------
for ($w = 0; $w < $workers; $w++) {
    $from = $w * $chunkSize + 2;
    $to   = min(($w + 1) * $chunkSize + 1, $limit);

    spawn(function() use ($from, $to, $chan) {
        $cnt = 0;
        for ($i = $from; $i <= $to; $i++) {
            $isPrime = true;
            for ($j = 2; $j * $j <= $i; $j++) {
                if ($i % $j == 0) {
                    $isPrime = false;
                    break;
                }
            }
            if ($isPrime) {
                $cnt++;
            }
        }
        // send partial count back to main thread
        send($chan, $cnt);
    });
}

// -----------------------------------------------------------------
// Collect all partial counts from workers
// -----------------------------------------------------------------
$totalPrimes = 0;
for ($w = 0; $w < $workers; $w++) {
    $partial      = receive($chan);
    $totalPrimes += $partial;
}

$end     = microtime(true);
$elapsed = $end - $start;

echo "Target Limit    : " . $limit . "\n";
echo "Workers         : " . $workers . "\n";
echo "Primes Found    : " . $totalPrimes . "\n";
echo "Execution Time  : " . $elapsed . " seconds\n";
echo "----------------------------------------\n";
?>
