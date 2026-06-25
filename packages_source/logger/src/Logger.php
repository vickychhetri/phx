<?php
class Logger {
    public function info($msg) {
        echo "[INFO] " . $msg . "\n";
    }

    public function success($msg) {
        echo "[SUCCESS] " . $msg . "\n";
    }

    public function warn($msg) {
        echo "[WARNING] " . $msg . "\n";
    }

    public function error($msg) {
        echo "[ERROR] " . $msg . "\n";
    }
}
?>
