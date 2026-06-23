<?php
use PHX\MySQL;
use PHX\Exception;

echo "==================================================\n";
echo "         PHX MySQL Database Demo (properly)       \n";
echo "==================================================\n";

$host = "127.0.0.1";
$user = "root";
$password = "MyNewPass@123";
$database = "app_production";

try {
    $db = new MySQL();
    
    echo "1. Connecting to MySQL server...\n";
    $db->connect($host, $user, $password, $database);
    
    // --- Step 2: Running Default Schema Creation ---
    echo "\n2. Initializing Default Database Schema...\n";
    
    echo " - Dropping existing tables if any...\n";
    $db->exec("DROP TABLE IF EXISTS transactions");
    $db->exec("DROP TABLE IF EXISTS customers");
    $db->exec("DROP TABLE IF EXISTS products");
    
    echo " - Creating table: customers\n";
    $db->exec("CREATE TABLE customers (id INT PRIMARY KEY, name VARCHAR(255) NOT NULL, email VARCHAR(255) NOT NULL)");
    
    echo " - Creating table: products\n";
    $db->exec("CREATE TABLE products (id INT PRIMARY KEY, name VARCHAR(255) NOT NULL, sku VARCHAR(50) NOT NULL, price INT NOT NULL)");
    
    echo " - Creating table: transactions\n";
    $db->exec("CREATE TABLE transactions (id INT PRIMARY KEY, customer_id INT NOT NULL, product_id INT NOT NULL, quantity INT NOT NULL)");
    
    // --- Step 3: Seeding Default Schema Data ---
    echo "\n3. Seeding Default Schema Data...\n";
    
    echo " - Inserting customers...\n";
    $db->exec("INSERT INTO customers (id, name, email) VALUES (1, 'Vicky Kumar', 'vicky@example.com')");
    $db->exec("INSERT INTO customers (id, name, email) VALUES (2, 'John Doe', 'john@example.com')");
    $db->exec("INSERT INTO customers (id, name, email) VALUES (3, 'Alice Smith', 'alice@example.com')");
    
    echo " - Inserting products...\n";
    $db->exec("INSERT INTO products (id, name, sku, price) VALUES (101, 'MacBook Pro', 'MBP14', 1999)");
    $db->exec("INSERT INTO products (id, name, sku, price) VALUES (102, 'iPhone 15', 'IPH15', 999)");
    $db->exec("INSERT INTO products (id, name, sku, price) VALUES (103, 'AirPods Pro', 'APP2', 249)");
    
    echo " - Inserting transactions...\n";
    $db->exec("INSERT INTO transactions (id, customer_id, product_id, quantity) VALUES (5001, 1, 101, 1)");
    $db->exec("INSERT INTO transactions (id, customer_id, product_id, quantity) VALUES (5002, 2, 103, 2)");
    $db->exec("INSERT INTO transactions (id, customer_id, product_id, quantity) VALUES (5003, 3, 102, 1)");
    
    // --- Step 4: Querying Database Tables ---
    echo "\n4. Querying Seeded Customers Table:\n";
    $customers = $db->query("SELECT * FROM customers");
    $custCount = count($customers);
    for ($i = 0; $i < $custCount; $i++) {
        $c = $customers[$i];
        echo "   [Customer] ID: " . $c['id'] . " | Name: " . $c['name'] . " | Email: " . $c['email'] . "\n";
    }
    
    echo "\n5. Querying Seeded Products Table:\n";
    $products = $db->query("SELECT * FROM products");
    $prodCount = count($products);
    for ($i = 0; $i < $prodCount; $i++) {
        $p = $products[$i];
        echo "   [Product] ID: " . $p['id'] . " | Name: " . $p['name'] . " | SKU: " . $p['sku'] . " | Price: $" . $p['price'] . "\n";
    }
    
    echo "\n6. Querying specific transactions WHERE customer_id = 1:\n";
    $vickyTx = $db->query("SELECT * FROM transactions WHERE customer_id = 1");
    $txCount = count($vickyTx);
    for ($i = 0; $i < $txCount; $i++) {
        $tx = $vickyTx[$i];
        echo "   [Transaction] TxID: " . $tx['id'] . " | Customer ID: " . $tx['customer_id'] . " | Product ID: " . $tx['product_id'] . " | Qty: " . $tx['quantity'] . "\n";
    }
    
    echo "\n7. Closing Database Connection...\n";
    $db->close();
    
} catch (Exception $e) {
    echo "MySQL Runtime Error: " . $e->getMessage() . "\n";
}

echo "==================================================\n";
?>
