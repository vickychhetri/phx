<?php
// -------------------------------------------------------
// Example 05: Functions & Closures
// -------------------------------------------------------

// Regular function
function greet($name) {
    return "Hello, " . $name . "!";
}
echo greet("World") . "\n";

// Function with default parameter
function power($base, $exp = 2) {
    $result = 1;
    for ($i = 0; $i < $exp; $i++) {
        $result = $result * $base;
    }
    return $result;
}
echo "2^8  = " . power(2, 8) . "\n";
echo "5^2  = " . power(5) . "\n";

// Recursive function
function factorial($n) {
    if ($n <= 1) {
        return 1;
    }
    return $n * factorial($n - 1);
}
echo "10!  = " . factorial(10) . "\n";

// Fibonacci
function fibonacci($n) {
    if ($n <= 1) {
        return $n;
    }
    return fibonacci($n - 1) + fibonacci($n - 2);
}
echo "fib(10) = " . fibonacci(10) . "\n";

// Anonymous function (closure)
$multiply = function($x, $y) {
    return $x * $y;
};
echo "7 * 6 = " . $multiply(7, 6) . "\n";

// Closure with `use`
$base = 100;
$addBase = function($x) use ($base) {
    return $base + $x;
};
echo "100 + 25 = " . $addBase(25) . "\n";
?>
