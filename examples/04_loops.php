<?php
// -------------------------------------------------------
// Example 04: Loops – while / do-while / for
// -------------------------------------------------------

echo "--- while loop ---\n";
$i = 1;
while ($i <= 5) {
    echo "i = " . $i . "\n";
    $i++;
}

echo "\n--- do-while loop ---\n";
$n = 1;
do {
    echo "n = " . $n . "\n";
    $n++;
} while ($n <= 5);

echo "\n--- for loop ---\n";
for ($x = 0; $x < 5; $x++) {
    echo "x = " . $x . "\n";
}

echo "\n--- for loop with break and continue ---\n";
for ($k = 1; $k <= 10; $k++) {
    if ($k % 2 == 0) {
        continue;
    }
    if ($k > 7) {
        break;
    }
    echo "k = " . $k . "\n";
}

echo "\n--- iterate array with for ---\n";
$fruits = ["Apple", "Banana", "Cherry", "Date"];
for ($f = 0; $f < count($fruits); $f++) {
    echo "Fruit: " . $fruits[$f] . "\n";
}

echo "\n--- iterate associative array keys ---\n";
$prices = ["Apple" => 1.5, "Banana" => 0.5, "Cherry" => 2.0];
$keys   = ["Apple", "Banana", "Cherry"];
for ($k = 0; $k < count($keys); $k++) {
    $key = $keys[$k];
    echo $key . " costs " . $prices[$key] . "\n";
}
?>
