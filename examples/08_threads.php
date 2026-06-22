<?php
// -------------------------------------------------------
// Example 08: Threads with spawn() and join
// -------------------------------------------------------

echo "--- Basic Threading ---\n";

// Spawn a background thread that prints a message
$t1 = spawn(function() {
    echo "Thread 1: Hello from goroutine!\n";
});

$t2 = spawn(function() {
    echo "Thread 2: Running in parallel!\n";
});

// Wait for both threads to complete
$t1->join();
$t2->join();

echo "Main: All threads finished.\n";

// -------------------------------------------------------
// Threads passing data back via return values
// -------------------------------------------------------

echo "\n--- Threads with return values ---\n";

function computeSum($from, $to) {
    $sum = 0;
    for ($i = $from; $i <= $to; $i++) {
        $sum += $i;
    }
    return $sum;
}

$threadA = spawn(function() {
    return computeSum(1, 500);
});

$threadB = spawn(function() {
    return computeSum(501, 1000);
});

$sumA = $threadA->join();
$sumB = $threadB->join();

echo "Sum(1..500)    = " . $sumA . "\n";
echo "Sum(501..1000) = " . $sumB . "\n";
echo "Total          = " . ($sumA + $sumB) . "\n";
?>
