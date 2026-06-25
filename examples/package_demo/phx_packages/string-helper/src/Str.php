<?php
class StringHelper {
    public function upper($str) {
        return strtoupper($str);
    }

    public function lower($str) {
        return strtolower($str);
    }

    public function length($str) {
        return strlen($str);
    }

    public function reverse($str) {
        $len = strlen($str);
        $res = "";
        for ($i = $len - 1; $i >= 0; $i--) {
            $res = $res . substr($str, $i, 1);
        }
        return $res;
    }
}
?>
