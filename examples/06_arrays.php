<?php
// -------------------------------------------------------
// Example 06: Arrays (using for loops for compatibility)
// -------------------------------------------------------

// Sequential array
$nums = [10, 20, 30, 40, 50];
echo "Count : " . count($nums) . "\n";

echo "Items:\n";
for ($i = 0; $i < count($nums); $i++) {
    echo "  " . $nums[$i] . "\n";
}

// Append to array
$nums[] = 60;
echo "After append, count = " . count($nums) . "\n";

// Associative array
$person = ["name" => "Vicky", "age" => 25, "lang" => "PHX"];

echo "\nName : " . $person["name"] . "\n";
echo "Age  : " . $person["age"] . "\n";
echo "Lang : " . $person["lang"] . "\n";

// Modify a key
$person["age"] = 26;
echo "New Age: " . $person["age"] . "\n";

// Nested array (matrix)
$matrix = [
    [1, 2, 3],
    [4, 5, 6],
    [7, 8, 9],
];

echo "\nMatrix:\n";
for ($r = 0; $r < 3; $r++) {
    for ($c = 0; $c < 3; $c++) {
        echo $matrix[$r][$c] . " ";
    }
    echo "\n";
}

// print_r of array
echo "\nprint_r:\n";
print_r($person);
?>
