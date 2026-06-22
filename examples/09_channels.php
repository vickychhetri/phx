<?php
// -------------------------------------------------------
// Example 09: Channels for Thread Communication
// -------------------------------------------------------

echo "--- Producer / Consumer via Channel ---\n";

// Buffered channel large enough for all messages
$chan = channel(10);

// Producer thread: generates numbers and sends them
$producer = spawn(function() use ($chan) {
    for ($i = 1; $i <= 5; $i++) {
        echo "Producer: sending " . $i . "\n";
        send($chan, $i);
    }
    // Send sentinel value to signal done
    send($chan, -1);
});

// Wait for producer to finish filling channel
$producer->join();

// Consumer in main thread
while (true) {
    $val = receive($chan);
    if ($val == -1) {
        echo "Consumer: done.\n";
        break;
    }
    echo "Consumer: received " . $val . "\n";
}

// -------------------------------------------------------
// Parallel prime count with 4 workers
// -------------------------------------------------------

echo "\n--- Parallel Prime Count (4 workers) ---\n";

$limit   = 100000;
$workers = 4;
$results = channel($workers);
$chunk   = intdiv($limit - 1, $workers) + 1;

for ($w = 0; $w < $workers; $w++) {
    $from = $w * $chunk + 2;
    $to   = min(($w + 1) * $chunk + 1, $limit);

    spawn(function() use ($from, $to, $results) {
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
        send($results, $cnt);
    });
}

$total = 0;
for ($w = 0; $w < $workers; $w++) {
    $total += receive($results);
}

echo "Primes up to " . $limit . ": " . $total . "\n";
?>
