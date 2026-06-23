<?php
// Include the services
include "project_demo/services/LogService.php";
include "project_demo/services/MathService.php";
include "project_demo/services/UserService.php";

logMessage("info", "Starting PHX Sample Project Demo");

// Call MathService
$limit = 1000;
logMessage("info", "Calculating sum of 1 to " . $limit);
$sum = calculateSum($limit);
logMessage("success", "Sum result: " . $sum);

$num = 5;
logMessage("info", "Calculating factorial of " . $num);
$fact = calculateFactorial($num);
logMessage("success", "Factorial result: " . $fact);

// Call UserService
logMessage("info", "Retrieving user information...");
$userStr = formatUserData(101, "Vicky Kumar", "vicky@example.com");
logMessage("success", "User Info: " . $userStr);

logMessage("info", "PHX Sample Project finished successfully.");
?>
