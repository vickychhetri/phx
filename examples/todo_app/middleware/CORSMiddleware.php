<?php
// middleware/CORSMiddleware.php

class CORSMiddleware {
    public function handle($req, $res, $next) {
        $res->header("Access-Control-Allow-Origin", "*")
            ->header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
            ->header("Access-Control-Allow-Headers", "Content-Type, Authorization, Cookie");
            
        if ($req->getMethod() === "OPTIONS") {
            $res->status(200)->end();
            return null;
        }
        
        return $next($req);
    }
}
?>
