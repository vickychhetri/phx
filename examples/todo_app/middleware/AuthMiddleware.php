<?php
// middleware/AuthMiddleware.php

class AuthMiddleware {
    public function handle($req, $res, $next) {
        $path = $req->getPath();
        
        // Exclude authorization pages from check
        $isAuthPath = false;
        if ($path === "/login") {
            $isAuthPath = true;
        }
        if ($path === "/signup") {
            $isAuthPath = true;
        }
        if ($path === "/logout") {
            $isAuthPath = true;
        }
        
        if ($isAuthPath) {
            return $next($req);
        }
        
        // Verify current user session
        $userId = get_current_user_id($req);
        if ($userId === null) {
            // Unauthenticated! Redirect to login page
            $res->status(302)->header("Location", "/login")->end();
            return null;
        }
        
        return $next($req);
    }
}
?>
