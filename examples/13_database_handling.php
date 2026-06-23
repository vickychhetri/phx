<?php
use PHX\DB;
use PHX\File;
use PHX\Exception;

echo "----------------------------------------\n";
echo "       PHX Database Package (DB)        \n";
echo "----------------------------------------\n";

$dbFile = "test_app.db";

try {
    $db = new DB();
    
    echo "Opening database file: " . $dbFile . "...\n";
    $db->open($dbFile);
    
    echo "Creating table 'users'...\n";
    $db->exec("CREATE TABLE users");
    
    echo "Inserting user records...\n";
    $db->exec("INSERT INTO users (id, name, role) VALUES (1, 'Vicky', 'Admin')");
    $db->exec("INSERT INTO users (id, name, role) VALUES (2, 'John', 'Developer')");
    $db->exec("INSERT INTO users (id, name, role) VALUES (3, 'Alice', 'Designer')");
    
    echo "Querying all users:\n";
    $users = $db->query("SELECT * FROM users");
    
    $count = count($users);
    echo "Found " . $count . " users:\n";
    for ($i = 0; $i < $count; $i++) {
        $user = $users[$i];
        echo " - ID: " . $user['id'] . " | Name: " . $user['name'] . " | Role: " . $user['role'] . "\n";
    }
    
    echo "\nQuerying specific user where name = 'Vicky':\n";
    $vickyResult = $db->query("SELECT * FROM users WHERE name = 'Vicky'");
    if (count($vickyResult) > 0) {
        $vicky = $vickyResult[0];
        echo " - Found: ID=" . $vicky['id'] . ", Role=" . $vicky['role'] . "\n";
    }
    
    echo "\nDeleting user ID 2 (John)...\n";
    $db->exec("DELETE FROM users WHERE id = 2");
    
    echo "Querying remaining users:\n";
    $remaining = $db->query("SELECT * FROM users");
    $remCount = count($remaining);
    for ($i = 0; $i < $remCount; $i++) {
        $u = $remaining[$i];
        echo " - ID: " . $u['id'] . " | Name: " . $u['name'] . " | Role: " . $u['role'] . "\n";
    }
    
    $db->close();
    echo "\nDatabase connection closed.\n";
    
    // Clean up database file
    $file = new File();
    $file->delete($dbFile);
    echo "Database file cleaned up.\n";

} catch (Exception $e) {
    echo "Database operation failed: " . $e->getMessage() . "\n";
}

echo "----------------------------------------\n";
?>
