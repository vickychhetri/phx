<?php

function hello() {
	return 1;
}

function goodbye() {
	return 2000;
}

function test(){
	$x = 1 + 2 + 3 + 4 + 5 + 6 + 7 + 8 + 9 + 10;
	echo "$x\n";
	$y = 0;
	for ($i = 1; $i <= 10; $i++){
		$y = $y + $i;
	}
	echo "$y\n";
	return 42;
}

$a = test();
echo "$a\n";


?>