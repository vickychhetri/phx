<?php
// controllers/AuthController.php

class AuthController {
    public function showLogin($req, $res) {
        $query = $req->getQuery();
        $error = "";
        $success = "";
        
        if (count($query) > 0) {
            if ($query["error"] !== null) {
                $error = $query["error"];
            }
            if ($query["success"] !== null) {
                $success = $query["success"];
            }
        }
        
        $view = new LoginView();
        $html = $view->render($error, $success);
        $res->status(200)->header("Content-Type", "text/html")->end($html);
    }
    
    public function login($req, $res) {
        $post = $req->getPost();
        $username = "";
        $password = "";
        
        if (count($post) > 0) {
            if ($post["username"] !== null) {
                $username = trim($post["username"]);
            }
            if ($post["password"] !== null) {
                $password = trim($post["password"]);
            }
        }
        
        $isEmpty = false;
        if ($username === "") {
            $isEmpty = true;
        }
        if ($password === "") {
            $isEmpty = true;
        }
        
        if ($isEmpty) {
            $res->status(302)->header("Location", "/login?error=Username+and+password+are+required")->end();
            return null;
        }
        
        $db = get_db_connection();
        $usernameEscaped = db_escape($username);
        
        try {
            $rows = $db->query("SELECT id, username, password FROM users WHERE username = '" . $usernameEscaped . "'");
            $db->close();
            
            if (count($rows) === 0) {
                $res->status(302)->header("Location", "/login?error=Invalid+username+or+password")->end();
                return null;
            }
            
            $user = $rows[0];
            if ($user["password"] !== $password) {
                $res->status(302)->header("Location", "/login?error=Invalid+username+or+password")->end();
                return null;
            }
            
            // Create session
            $token = microtime(true) . "_" . $user["id"];
            $cache = new Cache();
            $cache->set("session:" . $token, $user["id"], 3600); // 1 hour TTL
            
            $res->status(302)
                ->header("Set-Cookie", "session_token=" . $token . "; Path=/; HttpOnly")
                ->header("Location", "/dashboard")
                ->end();
        } catch (Exception $e) {
            $res->status(302)->header("Location", "/login?error=Database+query+failed")->end();
        }
    }
    
    public function showSignup($req, $res) {
        $query = $req->getQuery();
        $error = "";
        
        if (count($query) > 0) {
            if ($query["error"] !== null) {
                $error = $query["error"];
            }
        }
        
        $view = new SignupView();
        $html = $view->render($error);
        $res->status(200)->header("Content-Type", "text/html")->end($html);
    }
    
    public function signup($req, $res) {
        $post = $req->getPost();
        $username = "";
        $password = "";
        
        if (count($post) > 0) {
            if ($post["username"] !== null) {
                $username = trim($post["username"]);
            }
            if ($post["password"] !== null) {
                $password = trim($post["password"]);
            }
        }
        
        $isEmpty = false;
        if ($username === "") {
            $isEmpty = true;
        }
        if ($password === "") {
            $isEmpty = true;
        }
        
        if ($isEmpty) {
            $res->status(302)->header("Location", "/signup?error=All+fields+are+required")->end();
            return null;
        }
        
        $db = get_db_connection();
        $usernameEscaped = db_escape($username);
        $passwordEscaped = db_escape($password);
        
        try {
            $existing = $db->query("SELECT id FROM users WHERE username = '" . $usernameEscaped . "'");
            if (count($existing) > 0) {
                $db->close();
                $res->status(302)->header("Location", "/signup?error=Username+already+taken")->end();
                return null;
            }
            
            $db->exec("INSERT INTO users (username, password) VALUES ('" . $usernameEscaped . "', '" . $passwordEscaped . "')");
            $db->close();
            
            $res->status(302)->header("Location", "/login?success=Account+created+successfully.+Please+sign+in.")->end();
        } catch (Exception $e) {
            $db->close();
            $res->status(302)->header("Location", "/signup?error=Could+not+register+user")->end();
        }
    }
    
    public function logout($req, $res) {
        $token = get_session_token($req);
        if ($token !== "") {
            $cache = new Cache();
            $cache->set("session:" . $token, null, -1); // Expire immediately
        }
        
        $res->status(302)
            ->header("Set-Cookie", "session_token=; Path=/; Expires=Thu, 01 Jan 1970 00:00:00 GMT; HttpOnly")
            ->header("Location", "/login?success=Logged+out+successfully")
            ->end();
    }
}
?>
