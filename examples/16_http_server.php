<?php
// -------------------------------------------------------
// Example 16: High-Performance Event-Based HTTP Server
// -------------------------------------------------------

$server = new HttpServer();

// 1. Configure server settings
$server->configure([
    'environment' => 'production',
    'workers' => 8,
    'max_connections' => 10000,
    'keep_alive' => 60,
    'cors' => ['*'],
    'rate_limit' => 1000
]);

// 2. Register logging/metrics middleware
$server->use(function($req, $res, $next) {
    $method = $req->getMethod();
    $path = $req->getPath();
    $start = microtime(true);
    
    // Invoke next middleware in the pipeline
    $nextRes = $next($req);
    
    $duration = (microtime(true) - $start) * 1000;
    echo "[" . $method . "] " . $path . " - processed in " . $duration . "ms\n";
    return $nextRes;
});

// 3. Register custom dummy Auth middleware
$server->use(function($req, $res, $next) {
    // If accessing secret endpoint without token, return 401
    $path = $req->getPath();
    if ($path === "/api/v1/secret") {
        $headers = $req->getHeaders();
        $auth = "";
        if (count($headers) > 0) {
            // Find authorization header case-insensitively or check directly
            $auth = $headers["Authorization"];
        }
        if ($auth !== "Bearer PHXToken123") {
            echo "Auth Middleware: Blocked request to /api/v1/secret\n";
            $res->status(401)->json([
                "error" => "Unauthorized",
                "message" => "Invalid or missing Bearer token. Use Bearer PHXToken123."
            ]);
            return null; 
        }
    }
    return $next($req);
});

// 4. Configure local cache client
$server->cache([
    'driver' => 'memory',
    'ttl' => 300
]);

// 5. Configure database pools (MySQL and Postgres pools)
$server->database([
    'default' => [
        'driver' => 'mysql',
        'host' => 'localhost:3306',
        'database' => 'phx_db',
        'user' => 'root',
        'password' => '',
        'pool_size' => 15
    ]
]);

// 6. Define Router and Routes
$router = $server->router();

// Root route
$router->get('/', function($req, $res) {
    $res->status(200)
        ->header("Content-Type", "text/html")
        ->end("<h1>Welcome to PHX Advanced Cloud Services!</h1><p>High-performance event-based microservices with middleware, routing, and database pools.</p>");
});

// API group routes
$router->group('/api/v1', function($group) {
    // Basic JSON status endpoint
    $group->get('/status', function($req, $res) {
        $res->json([
            "status" => "healthy",
            "runtime" => "PHX Engine 1.0",
            "uptime" => microtime(true)
        ]);
    });

    // Dynamic path parameters endpoint
    $group->get('/users/{id}', function($req, $res, $id) {
        $res->json([
            "user_id" => $id,
            "name" => "PHX Developer #" . $id,
            "roles" => ["developer", "architect"]
        ]);
    });

    // Dynamic nested parameters endpoint
    $group->get('/users/{user_id}/posts/{post_id}', function($req, $res, $userId, $postId) {
        $res->json([
            "user_id" => $userId,
            "post_id" => $postId,
            "title" => "Concurrently Handling Millions of Requests with PHX",
            "body" => "Go-backed event polling provides unmatched throughput."
        ]);
    });

    // POST body echo endpoint
    $group->post('/echo', function($req, $res) {
        $body = $req->getBody();
        $res->json([
            "received_body" => $body,
            "timestamp" => microtime(true)
        ]);
    });

    // Cache test endpoint (Set/Get)
    $group->get('/cache-test', function($req, $res) {
        // Retrieve global cache instance
        $cache = $req->getParam("unused"); // Not needed, just retrieve cache from request context or globally
        // We can access cache directly or via a wrapper helper
        // Since cache is registered globally, we can use new Cache() to access it or configure it
        $c = new Cache();
        
        $hits = $c->get("hits");
        if ($hits === null) {
            $hits = 1;
        } else {
            $hits = $hits + 1;
        }
        $c->set("hits", $hits, 10); // Expire in 10s
        
        $res->json([
            "cache_hits" => $hits,
            "message" => "This counter is cached in memory for 10 seconds"
        ]);
    });

    // Protected endpoint
    $group->get('/secret', function($req, $res) {
        $res->json([
            "secret_data" => "PHX is faster than Node.js!",
            "authorized" => true
        ]);
    });
});

// Start the server
$server->listen(8085);
?>
