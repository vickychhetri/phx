<?php
// config/database.php

$db_host = "127.0.0.1:3306";
$db_user = "root";
$db_pass = "MyNewPass@123";
$db_name = "todo_app";

// 1. Auto-migration / DB creation routine
try {
    $db = new MySQL();
    echo "Connecting to MySQL at " . $db_host . " for migration checks...\n";
    $db->connect($db_host, $db_user, $db_pass, $db_name);
    
    // Create users table
    echo "Ensuring users table exists...\n";
    $db->exec("CREATE TABLE IF NOT EXISTS users (
        id INT AUTO_INCREMENT PRIMARY KEY,
        username VARCHAR(255) UNIQUE NOT NULL,
        password VARCHAR(255) NOT NULL,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    ) ENGINE=InnoDB");
    
    // Create todos table
    echo "Ensuring todos table exists...\n";
    $db->exec("CREATE TABLE IF NOT EXISTS todos (
        id INT AUTO_INCREMENT PRIMARY KEY,
        user_id INT NOT NULL,
        title VARCHAR(255) NOT NULL,
        completed BOOLEAN NOT NULL DEFAULT FALSE,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
    ) ENGINE=InnoDB");
    
    $db->close();
    echo "MySQL database and tables successfully verified.\n";
} catch (Exception $e) {
    echo "Database Migration / Creation Error: " . $e->getMessage() . "\n";
}

// 2. Helper function to escape strings for raw query construction
function db_escape($str) {
    return str_replace("'", "''", $str);
}

// 2b. Helper to obtain a fresh MySQL connection per request
function get_db_connection() {
    $conn = new MySQL();
    $conn->connect("127.0.0.1:3306", "root", "MyNewPass@123", "todo_app");
    return $conn;
}

// 3. Helper to parse cookies from request headers
function get_session_token($req) {
    $headers = $req->getHeaders();
    $cookieHeader = "";
    if (count($headers) > 0) {
        if ($headers["Cookie"] !== null) {
            $cookieHeader = $headers["Cookie"];
        } else if ($headers["cookie"] !== null) {
            $cookieHeader = $headers["cookie"];
        }
    }
    
    if ($cookieHeader === "") {
        return "";
    }
    
    // Find starting position of session_token=
    $pos = strpos($cookieHeader, "session_token=");
    if ($pos === false) {
        return "";
    }
    
    $tokenPart = substr($cookieHeader, $pos + 14);
    $semi = strpos($tokenPart, ";");
    if ($semi !== false) {
        return substr($tokenPart, 0, $semi);
    }
    return $tokenPart;
}

// 4. Helper to resolve current authenticated user ID
function get_current_user_id($req) {
    $token = get_session_token($req);
    if ($token === "") {
        return null;
    }
    $cache = new Cache();
    return $cache->get("session:" . $token);
}

// 5. Helper to resolve current user object/username from DB
function get_current_username($req, $server) {
    $userId = get_current_user_id($req);
    if ($userId === null) {
        return "";
    }
    
    try {
        $db = $server->db("default");
        $rows = $db->query("SELECT username FROM users WHERE id = " . $userId);
        if (count($rows) > 0) {
            $user = $rows[0];
            return $user["username"];
        }
    } catch (Exception $e) {
        echo "Error fetching current username: " . $e->getMessage() . "\n";
    }
    return "";
}
?>
