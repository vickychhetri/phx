<?php
include 'lib/cpu.php';
include 'lib/memory.php';

use System\CPU\Monitor as CPUMonitor;
use System\Memory\Monitor as MemoryMonitor;

$cpu = new CPUMonitor();
$mem = new MemoryMonitor();

echo "--- PHX System Monitor ---\n";
echo "CPU Usage:    " . $cpu->getUsage() . "%\n";
echo "Memory Usage: " . $mem->getUsage() . "%\n";
echo "--------------------------\n";
?>
