<?php
use PHX\Exception;

echo "----------------------------------------\n";
echo "    PHX Error Handling & Exceptions     \n";
echo "----------------------------------------\n";

function divide($a, $b) {
    if ($b == 0) {
        throw new Exception("Division by zero!");
    }
    return $a / $b;
}

try {
    echo "Attempting to divide 10 by 2...\n";
    $result = divide(10, 2);
    echo "Result: " . $result . "\n\n";

    echo "Attempting to divide 10 by 0...\n";
    $result = divide(10, 0);
    echo "Result: " . $result . "\n";
} catch (Exception $e) {
    echo "Caught exception: " . $e->getMessage() . "\n";
}

echo "----------------------------------------\n";
?>
