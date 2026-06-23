<?php
function calculateFactorial($n) {
    $result = 1;
    for ($i = 1; $i <= $n; $i++) {
        $result = $result * $i;
    }
    return $result;
}

function calculateSum($limit) {
    $sum = 0;
    for ($i = 1; $i <= $limit; $i++) {
        $sum += $i;
    }
    return $sum;
}
?>
