<?php
// controllers/TodoController.php

class TodoController {
    public function index($req, $res) {
        $userId = get_current_user_id($req);
        if ($userId === null) {
            $res->status(302)->header("Location", "/login")->end();
            return null;
        }
        
        $db = get_db_connection();
        
        try {
            // Fetch username
            $userRows = $db->query("SELECT username FROM users WHERE id = " . $userId);
            $username = "User";
            if (count($userRows) > 0) {
                $u = $userRows[0];
                $username = $u["username"];
            }
            
            // Fetch todos for this user
            $todos = $db->query("SELECT id, title, completed FROM todos WHERE user_id = " . $userId . " ORDER BY created_at DESC");
            $db->close();
            
            $view = new DashboardView();
            $html = $view->render($todos, $username);
            $res->status(200)->header("Content-Type", "text/html")->end($html);
        } catch (Exception $e) {
            $db->close();
            $res->status(500)->header("Content-Type", "text/html")->end("<h1>Internal Server Error</h1><p>" . $e->getMessage() . "</p>");
        }
    }

    public function store($req, $res) {
        $userId = get_current_user_id($req);
        if ($userId === null) {
            $res->status(302)->header("Location", "/login")->end();
            return null;
        }
        
        $post = $req->getPost();
        $title = "";
        if (count($post) > 0) {
            if ($post["title"] !== null) {
                $title = trim($post["title"]);
            }
        }
        
        if ($title !== "") {
            $db = get_db_connection();
            $titleEscaped = db_escape($title);
            try {
                $db->exec("INSERT INTO todos (user_id, title, completed) VALUES (" . $userId . ", '" . $titleEscaped . "', false)");
                $db->close();
            } catch (Exception $e) {
                $db->close();
            }
        }
        
        $res->status(302)->header("Location", "/dashboard")->end();
    }

    public function toggle($req, $res, $id) {
        $userId = get_current_user_id($req);
        if ($userId === null) {
            $res->status(302)->header("Location", "/login")->end();
            return null;
        }
        
        $db = get_db_connection();
        try {
            // First fetch current completion state to invert it
            $rows = $db->query("SELECT completed FROM todos WHERE id = " . $id . " AND user_id = " . $userId);
            if (count($rows) > 0) {
                $t = $rows[0];
                $completedVal = $t["completed"];
                
                $newCompleted = "true";
                $isCompleted = false;
                if ($completedVal === true) {
                    $isCompleted = true;
                }
                if ($completedVal === 1) {
                    $isCompleted = true;
                }
                if ($completedVal === "1") {
                    $isCompleted = true;
                }
                if ($completedVal === "true") {
                    $isCompleted = true;
                }
                
                if ($isCompleted) {
                    $newCompleted = "false";
                }
                
                $db->exec("UPDATE todos SET completed = " . $newCompleted . " WHERE id = " . $id . " AND user_id = " . $userId);
            }
            $db->close();
        } catch (Exception $e) {
            $db->close();
        }
        
        $res->status(302)->header("Location", "/dashboard")->end();
    }

    public function destroy($req, $res, $id) {
        $userId = get_current_user_id($req);
        if ($userId === null) {
            $res->status(302)->header("Location", "/login")->end();
            return null;
        }
        
        $db = get_db_connection();
        try {
            $db->exec("DELETE FROM todos WHERE id = " . $id . " AND user_id = " . $userId);
            $db->close();
        } catch (Exception $e) {
            $db->close();
        }
        
        $res->status(302)->header("Location", "/dashboard")->end();
    }
}
?>
