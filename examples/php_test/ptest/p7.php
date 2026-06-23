<?php
// -------------------------------------------------
// PHX Channel‑Thread Benchmark
// -------------------------------------------------

echo "----------------------------------------\n";
echo "    PHX Channel‑Thread Performance\n";
echo "----------------------------------------\n";

$start   = microtime(true);
$limit   = 500000;          // upper bound for the sieve
$workers = 4;               // number of concurrent workers

// Create a channel that workers will use to send their partial counts
$chan = channel();

// Determine the size of each chunk (roughly equal work per worker)
$chunk = intdiv($limit - 1, $workers) + 1;

// ----------------------------------------------------------------
