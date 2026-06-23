<?php
echo "--- PHX CPU Benchmark (Fixed Safe Version) ---\n";
echo "Starting 4 workers...\n";

$workers = 4;
$limit = 50000000;
$tasks = channel(10);
$results = channel(10);

$start = microtime(true);

for ($w = 0; $w < $workers; $w++) {
    spawn(function() use ($tasks, $results, $w) {
        echo "Worker {$w}: started\n";
        $total = 0;
        while (true) {
            $task = receive($tasks);
            if ($task == -1) {
                echo "Worker {$w}: stop\n";
                break;
            }
            $from = $task[0];
            $to = $task[1];
            for ($i = $from; $i < $to; $i++) {
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
        }
        send($results, $total);
        echo "Worker {$w}: exited\n";
    });
}

// Send tasks dynamically based on limit and workers
$chunk = intdiv($limit, $workers);
$start_i = 0;
for ($w = 0; $w < $workers; $w++) {
    $from = $start_i;
    if ($w == $workers - 1) {
        $to = $limit;
    } else {
        $to = $start_i + $chunk;
    }
    echo "Main: sending {$from} -> {$to}\n";
    send($tasks, [$from, $to]);
    $start_i = $to;
}

// STOP signals
for ($w = 0; $w < $workers; $w++) {
    send($tasks, -1);
}

echo "Main: waiting results...\n";
$total = 0;
for ($w = 0; $w < $workers; $w++) {
    $total += receive($results);
}

$elapsed = microtime(true) - $start;
echo "Result: {$total}\n";
echo "Time: {$elapsed} sec\n";
?>
