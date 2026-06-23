<?php
echo "--- Arrays & Hash Maps ---\n";
$arr = [1, 2, "three"];
echo "Length of sequential array: " . count($arr) . "\n";
print_r($arr);

$hash = ["name" => "Vicky", "role" => "developer"];
echo "Name: " . $hash["name"] . "\n";
$hash["salary"] = 100000;
print_r($hash);

$append = [];
$append[] = "first";
$append[] = "second";
print_r($append);

echo "\n--- String Indexing ---\n";
$str = "Antigravity";
echo "First char: " . $str[0] . "\n";
echo "Last char: " . $str[10] . "\n";

echo "\n--- Standard Library ---\n";
echo "strlen: " . strlen("hello") . "\n";
echo "strpos: " . strpos("hello", "ell") . "\n";
echo "substr: " . substr("Antigravity", 4, 6) . "\n";
echo "max: " . max(5, 12, 3) . "\n";
echo "min: " . min(5, 12, 3) . "\n";
echo "intval: " . intval("123") . "\n";
?>
