<?php
use PHX\MySQL;
use PHX\PostgreSQL;
use PHX\MongoDB;
use PHX\Exception;

echo "========================================\n";
echo "    PHX Multi-Database Driver Demo      \n";
echo "========================================\n";

$host = "127.0.0.1";
$user = "root";
$password = "MyNewPass@123";

// 1. MySQL Operations
echo "\n--- [MySQL Client] ---\n";
try {
    $mysql = new MySQL();
    echo "Connecting to MySQL...\n";
    $mysql->connect($host, $user, $password, "ecom_db");
    
    echo "Creating table 'products'...\n";
    $mysql->exec("CREATE TABLE products");
    
    echo "Inserting products...\n";
    $mysql->exec("INSERT INTO products (id, name, price) VALUES (1, 'Laptop', 1200)");
    $mysql->exec("INSERT INTO products (id, name, price) VALUES (2, 'Smartphone', 800)");
    
    echo "Querying all products:\n";
    $products = $mysql->query("SELECT * FROM products");
    $pCount = count($products);
    for ($i = 0; $i < $pCount; $i++) {
        $p = $products[$i];
        echo " - Product: " . $p['name'] . " ($" . $p['price'] . ")\n";
    }
    
    $mysql->close();
} catch (Exception $e) {
    echo "MySQL Exception: " . $e->getMessage() . "\n";
}

// 2. PostgreSQL Operations
// echo "\n--- [PostgreSQL Client] ---\n";
// try {
//     $pg = new PostgreSQL();
//     echo "Connecting to PostgreSQL...\n";
//     $pg->connect($host, $user, $password, "customer_db", 5432);
    
//     echo "Creating table 'orders'...\n";
//     $pg->exec("CREATE TABLE orders");
    
//     echo "Inserting orders...\n";
//     $pg->exec("INSERT INTO orders (id, user_id, amount) VALUES (101, 1, 1200)");
//     $pg->exec("INSERT INTO orders (id, user_id, amount) VALUES (102, 2, 800)");
    
//     echo "Querying all orders:\n";
//     $orders = $pg->query("SELECT * FROM orders");
//     $oCount = count($orders);
//     for ($i = 0; $i < $oCount; $i++) {
//         $o = $orders[$i];
//         echo " - Order ID: " . $o['id'] . " | Amount: $" . $o['amount'] . "\n";
//     }
    
//     $pg->close();
// } catch (Exception $e) {
//     echo "PostgreSQL Exception: " . $e->getMessage() . "\n";
// }

// // 3. MongoDB Operations
// echo "\n--- [MongoDB Client] ---\n";
// try {
//     $mongo = new MongoDB();
//     $uri = "mongodb://" . $user . ":" . $password . "@" . $host . "/admin";
//     echo "Connecting to MongoDB via URI: " . $uri . "\n";
//     $mongo->connect($uri);
    
//     $mongo->selectDatabase("analytics");
//     $mongo->selectCollection("logs");
    
//     echo "Inserting document logs...\n";
//     $mongo->insertOne(["id" => 1, "level" => "INFO", "message" => "Server started successfully"]);
//     $mongo->insertOne(["id" => 2, "level" => "ERROR", "message" => "Database connection failed"]);
//     $mongo->insertOne(["id" => 3, "level" => "INFO", "message" => "User logged in"]);
    
//     echo "Finding logs with level = 'INFO':\n";
//     $infoLogs = $mongo->find(["level" => "INFO"]);
//     $lCount = count($infoLogs);
//     for ($i = 0; $i < $lCount; $i++) {
//         $log = $infoLogs[$i];
//         echo " - Log ID: " . $log['id'] . " | Level: " . $log['level'] . " | Msg: " . $log['message'] . "\n";
//     }
    
//     $mongo->close();
// } catch (Exception $e) {
//     echo "MongoDB Exception: " . $e->getMessage() . "\n";
// }

echo "\n========================================\n";
?>
