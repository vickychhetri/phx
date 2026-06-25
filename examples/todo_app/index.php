<?php
// index.php
// Production-Ready PHX Todo Application Entrypoint

$server = new HttpServer();

// 1. Configure production-ready server settings
$server->configure([
    'environment' => 'production',
    'workers' => 8,
    'max_connections' => 10000,
    'keep_alive' => 60,
    'rate_limit' => 5000
]);

// 2. Include project files (ordered by dependencies)
// NOTE: If running or copying this project to another location,
// make sure the directory prefix below matches your execution path.
include "examples/todo_app/config/database.php";
include "examples/todo_app/middleware/CORSMiddleware.php";
include "examples/todo_app/middleware/AuthMiddleware.php";
include "examples/todo_app/views/layout.php";
include "examples/todo_app/views/login.php";
include "examples/todo_app/views/signup.php";
include "examples/todo_app/views/dashboard.php";
include "examples/todo_app/controllers/AuthController.php";
include "examples/todo_app/controllers/TodoController.php";

// 3. Register global middlewares in sequence
$server->use("CORSMiddleware");
$server->use("AuthMiddleware");

// 4. Configure database connection pools
$server->database([
    'default' => [
        'driver' => 'mysql',
        'host' => '127.0.0.1:3306',
        'database' => 'todo_app',
        'user' => 'root',
        'password' => 'MyNewPass@123',
        'pool_size' => 15
    ]
]);

// 5. Load and mount application routes
include "examples/todo_app/routes/web.php";

// 6. Start the server
echo "PHX Todo App starting at http://localhost:8085...\n";
$server->listen(8085);
?>
