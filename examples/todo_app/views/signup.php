<?php
// views/signup.php

class SignupView {
    public function render($error = "") {
        $alertHtml = "";
        if ($error !== "") {
            $alertHtml = '<div class="alert alert-danger">❌ ' . $error . '</div>';
        }
        
        $bodyHtml = '<div class="card">
            <div class="card-header">
                <h1 class="card-title">Create Account</h1>
                <p class="card-desc">Get started with our high-throughput event server</p>
            </div>
            
            ' . $alertHtml . '
            
            <form action="/signup" method="POST">
                <div class="form-group">
                    <label class="form-label" for="username">Username</label>
                    <input class="form-control" type="text" id="username" name="username" placeholder="Choose a username" required autocomplete="off">
                </div>
                
                <div class="form-group">
                    <label class="form-label" for="password">Password</label>
                    <input class="form-control" type="password" id="password" name="password" placeholder="Choose a strong password" required>
                </div>
                
                <button type="submit" class="btn-submit">Sign Up</button>
            </form>
            
            <div class="card-footer">
                Already have an account? <a href="/login">Sign In</a>
            </div>
        </div>';
        
        $layout = new Layout();
        return $layout->render("Sign Up", $bodyHtml, "");
    }
}
?>
