<?php
// -------------------------------------------------------
// Example 03: Control Flow – if / elseif / else / switch
// -------------------------------------------------------

$score = 78;

echo "Score: " . $score . "\n";

// if / elseif / else
if ($score >= 90) {
    echo "Grade: A\n";
} elseif ($score >= 75) {
    echo "Grade: B\n";
} elseif ($score >= 60) {
    echo "Grade: C\n";
} else {
    echo "Grade: F\n";
}

// Ternary
$status = ($score >= 60) ? "Pass" : "Fail";
echo "Status: " . $status . "\n";

// Switch
$day = 3;
echo "\nDay " . $day . " is: ";
switch ($day) {
    case 1:
        echo "Monday\n";
        break;
    case 2:
        echo "Tuesday\n";
        break;
    case 3:
        echo "Wednesday\n";
        break;
    case 4:
        echo "Thursday\n";
        break;
    case 5:
        echo "Friday\n";
        break;
    default:
        echo "Weekend\n";
        break;
}
?>
