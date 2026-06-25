<?php
class MathHelper {
    public function add($a, $b) {
        return $a + $b;
    }

    public function subtract($a, $b) {
        return $a - $b;
    }

    public function multiply($a, $b) {
        return $a * $b;
    }

    public function divide($a, $b) {
        if ($b == 0) {
            echo "[ERROR] Division by zero\n";
            return null;
        }
        return $a / $b;
    }

    public function factorial($n) {
        if ($n <= 1) {
            return 1;
        }
        return $n * $this->factorial($n - 1);
    }
}
?>
