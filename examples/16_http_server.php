<?php
// -------------------------------------------------------
// Example 16: High-Performance Event-Based HTTP Server
// -------------------------------------------------------

$server = new HttpServer();

$server->listen(8085, function($req, $res) {
    $method = $req->getMethod();
    $path = $req->getPath();
    
    echo "[" . $method . "] " . $path . "\n";
    
    // Router logic
    if ($path === "/") {
        $res->status(200)
            ->header("Content-Type", "text/html")
            ->end("<h1>Welcome to PHX Cloud Services!</h1><p>Running extremely efficiently on native Go runtime.</p>");
    } else if ($path === "/hello") {
        // Read query parameters
        $query = $req->getQuery();
        $name = "Guest";
        if (count($query) > 0) {
            $name = $query["name"];
        }
        $res->status(200)
            ->header("Content-Type", "text/plain")
            ->end("Hello " . $name . "! Welcome to the future of PHP.");
    } else if ($path === "/api/stats") {
        // Return JSON payload
        $res->json([
            "runtime" => "PHX v1.0",
            "language" => "PHP compiled to Go",
            "concurrency" => "Event-based epoll/goroutines",
            "status" => "healthy",
            "time" => microtime(true)
        ]);
    } else if ($path === "/echo") {
        if ($method === "POST") {
            // Echo back the request body and headers
            $body = $req->getBody();
            $headers = $req->getHeaders();
            $res->json([
                "body" => $body,
                "headers" => $headers,
                "success" => true
            ]);
        } else {
            $res->status(405)->end("Method Not Allowed");
        }
    } else {
        // 404 handler
        $res->status(404)
            ->header("Content-Type", "text/plain")
            ->end("Error 404: Endpoint Not Found");
    }
});
?>
