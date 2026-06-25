<?php
include "phx_packages/autoload.php";

$log = new Logger();
$log->info("Starting PHX package manager verification...");

$math = new MathHelper();
$a = 15;
$b = 5;

$strhelp = new StringHelper();
$str = "Hello, World!";
$length = $strhelp->length($str);
$log->success("StringHelper::length(\"" . $str . "\") = " . $length);

$sum = $math->add($a, $b);
$log->success("MathHelper::add(" . $a . ", " . $b . ") = " . $sum);

$diff = $math->subtract($a, $b);
$log->success("MathHelper::subtract(" . $a . ", " . $b . ") = " . $diff);

$prod = $math->multiply($a, $b);
$log->success("MathHelper::multiply(" . $a . ", " . $b . ") = " . $prod);

$div = $math->divide($a, $b);
$log->success("MathHelper::divide(" . $a . ", " . $b . ") = " . $div);

$n = 5;
$fact = $math->factorial($n);
$log->success("MathHelper::factorial(" . $n . ") = " . $fact);

$log->warn("Verification successfully completed!");
?>
