<?php
// routes/web.php

$router = $server->router();

// Root route handler with redirection logic based on auth status
$router->get("/", function($req, $res) {
    $userId = get_current_user_id($req);
    if ($userId !== null) {
        $res->status(302)->header("Location", "/dashboard")->end();
    } else {
        $res->status(302)->header("Location", "/login")->end();
    }
});

// Authentication routes
$router->get("/login", "AuthController@showLogin");
$router->post("/login", "AuthController@login");
$router->get("/signup", "AuthController@showSignup");
$router->post("/signup", "AuthController@signup");
$router->get("/logout", "AuthController@logout");

// Protected Todo dashboard and CRUD routes
$router->get("/dashboard", "TodoController@index");
$router->post("/todos/add", "TodoController@store");
$router->get("/todos/toggle/{id}", "TodoController@toggle");
$router->get("/todos/delete/{id}", "TodoController@destroy");
?>
