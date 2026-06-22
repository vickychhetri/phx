<?php
// -------------------------------------------------------
// Example 02: Variables, Operators & Math
// -------------------------------------------------------

$a = 42;
$b = 7;

echo "a = " . $a . ", b = " . $b . "\n";
echo "Addition       : " . ($a + $b) . "\n";
echo "Subtraction    : " . ($a - $b) . "\n";
echo "Multiplication : " . ($a * $b) . "\n";
echo "Division       : " . ($a / $b) . "\n";
echo "Modulo         : " . ($a % $b) . "\n";
echo "Integer Div    : " . intdiv($a, $b) . "\n";

// Float math
$pi = 3.14159;
$r  = 5.0;
echo "\nCircle radius  : " . $r . "\n";
echo "Area           : " . ($pi * $r * $r) . "\n";
echo "Circumference  : " . (2 * $pi * $r) . "\n";

// Comparison
echo "\n42 > 7  ? " . ($a > $b ? "Yes" : "No") . "\n";
echo "42 == 7 ? " . ($a == $b ? "Yes" : "No") . "\n";

// Abs / min / max
echo "\nabs(-15)      = " . abs(-15) . "\n";
echo "min(3, 9, 1)  = " . min(3, 9, 1) . "\n";
echo "max(3, 9, 1)  = " . max(3, 9, 1) . "\n";
?>
