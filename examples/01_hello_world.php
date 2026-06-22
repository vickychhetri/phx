<?php
// -------------------------------------------------------
// Example 01: Hello World & String Operations
// -------------------------------------------------------

echo "Hello, World!\n";

$name = "PHX";
$version = "1.0";

echo "Welcome to " . $name . " Runtime v" . $version . "\n";

// String interpolation
echo "Language: {$name}\n";

// String functions
$str = "The quick brown fox";
echo "Length     : " . strlen($str) . "\n";
echo "Uppercase  : " . strtoupper($str) . "\n";
echo "Substring  : " . substr($str, 4, 5) . "\n";
echo "Find 'fox' : " . strpos($str, "fox") . "\n";
?>
