<?php
// views/login.php

class LoginView {
    public function render($error = "", $success = "") {
        $alertHtml = "";
        if ($error !== "") {
            $alertHtml = '<div class="alert alert-danger">❌ ' . $error . '</div>';
        } else if ($success !== "") {
            $alertHtml = '<div class="alert alert-success">✅ ' . $success . '</div>';
        }
        
        $bodyHtml = '<div class="card">
            <div class="card-header">
                <h1 class="card-title">Welcome Back</h1>
                <p class="card-desc">Sign in to manage your high-performance task pipeline</p>
            </div>
            
            ' . $alertHtml . '
            
            <form action="/login" method="POST">
                <div class="form-group">
                    <label class="form-label" for="username">Username</label>
                    <input class="form-control" type="text" id="username" name="username" placeholder="Enter your username" required autocomplete="off">
                </div>
                
                <div class="form-group">
                    <label class="form-label" for="password">Password</label>
                    <input class="form-control" type="password" id="password" name="password" placeholder="Enter your password" required>
                </div>
                
                <button type="submit" class="btn-submit">Sign In</button>
            </form>
            
            <div class="card-footer">
                Don\'t have an account? <a href="/signup">Sign Up</a>
            </div>
        </div>';
        
        $layout = new Layout();
        return $layout->render("Login", $bodyHtml, "");
    }
}
?>
