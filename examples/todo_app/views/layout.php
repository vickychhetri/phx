<?php
// views/layout.php

class Layout {
    public function render($title, $bodyHtml, $currentUser = "") {
        $userSection = "";
        if ($currentUser !== "") {
            $userSection = '<div class="user-badge">
                <span class="user-icon">👤</span>
                <span class="username">' . $currentUser . '</span>
                <a href="/logout" class="btn-logout">Logout</a>
            </div>';
        } else {
            $userSection = '<div class="auth-links">
                <a href="/login" class="auth-link">Login</a>
                <a href="/signup" class="auth-link signup-btn">Sign Up</a>
            </div>';
        }

        $html = '<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>PHX - ' . $title . '</title>
    <!-- Google Fonts -->
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Outfit:wght@300;400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg-color: #030712;
            --card-bg: rgba(17, 24, 39, 0.7);
            --border-color: rgba(255, 255, 255, 0.08);
            --text-primary: #f3f4f6;
            --text-secondary: #9ca3af;
            --accent-purple: #8b5cf6;
            --accent-pink: #ec4899;
            --accent-grad: linear-gradient(135deg, #8b5cf6, #ec4899);
            --code-bg: #090d16;
            --success-color: #10b981;
            --danger-color: #ef4444;
        }

        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }

        body {
            font-family: "Outfit", sans-serif;
            background-color: var(--bg-color);
            color: var(--text-primary);
            min-height: 100vh;
            display: flex;
            flex-direction: column;
            padding: 20px;
            background-image: 
                radial-gradient(circle at 10% 20%, rgba(139, 92, 246, 0.08) 0%, transparent 40%),
                radial-gradient(circle at 90% 80%, rgba(236, 72, 153, 0.08) 0%, transparent 40%);
        }

        .navbar {
            display: flex;
            justify-content: space-between;
            align-items: center;
            width: 100%;
            max-width: 800px;
            margin: 0 auto 30px auto;
            padding: 15px 30px;
            background: var(--card-bg);
            border: 1px solid var(--border-color);
            border-radius: 20px;
            backdrop-filter: blur(10px);
        }

        .nav-logo {
            font-family: "JetBrains Mono", monospace;
            font-size: 1.5rem;
            font-weight: 700;
            background: var(--accent-grad);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            text-decoration: none;
            letter-spacing: -1px;
        }

        .user-badge {
            display: flex;
            align-items: center;
            gap: 12px;
            background: rgba(255, 255, 255, 0.05);
            padding: 8px 16px;
            border-radius: 12px;
            border: 1px solid var(--border-color);
        }

        .username {
            font-weight: 500;
            font-size: 0.9rem;
            color: #ffffff;
        }

        .btn-logout {
            color: var(--accent-pink);
            text-decoration: none;
            font-size: 0.85rem;
            font-weight: 600;
            transition: opacity 0.2s;
            margin-left: 8px;
            border-left: 1px solid var(--border-color);
            padding-left: 12px;
        }

        .btn-logout:hover {
            opacity: 0.8;
        }

        .auth-links {
            display: flex;
            align-items: center;
            gap: 15px;
        }

        .auth-link {
            color: var(--text-primary);
            text-decoration: none;
            font-size: 0.9rem;
            font-weight: 500;
            transition: color 0.2s;
        }

        .auth-link:hover {
            color: var(--accent-purple);
        }

        .signup-btn {
            background: var(--accent-grad);
            padding: 8px 18px;
            border-radius: 10px;
            color: #ffffff;
            font-weight: 600;
            box-shadow: 0 4px 10px rgba(139, 92, 246, 0.2);
        }

        .signup-btn:hover {
            color: #ffffff;
            transform: translateY(-1px);
            box-shadow: 0 6px 15px rgba(139, 92, 246, 0.4);
        }

        .main-content {
            flex-grow: 1;
            display: flex;
            justify-content: center;
            align-items: center;
            width: 100%;
        }

        .card {
            width: 100%;
            max-width: 500px;
            background-color: var(--card-bg);
            border: 1px solid var(--border-color);
            border-radius: 24px;
            padding: 40px;
            backdrop-filter: blur(20px);
            box-shadow: 0 10px 50px rgba(0, 0, 0, 0.4);
            animation: fadeIn 0.5s ease-out;
        }

        .dashboard-card {
            max-width: 800px;
        }

        @keyframes fadeIn {
            from { opacity: 0; transform: translateY(15px); }
            to { opacity: 1; transform: translateY(0); }
        }

        .card-header {
            text-align: center;
            margin-bottom: 30px;
        }

        .card-title {
            font-size: 1.8rem;
            font-weight: 700;
            margin-bottom: 8px;
            background: linear-gradient(135deg, #ffffff, #9ca3af);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }

        .card-desc {
            font-size: 0.9rem;
            color: var(--text-secondary);
        }

        /* Form Controls */
        .form-group {
            margin-bottom: 20px;
            display: flex;
            flex-direction: column;
            gap: 8px;
        }

        .form-label {
            font-size: 0.85rem;
            font-weight: 600;
            color: var(--text-secondary);
            text-transform: uppercase;
            letter-spacing: 1px;
        }

        .form-control {
            width: 100%;
            padding: 14px 18px;
            background-color: rgba(255, 255, 255, 0.03);
            border: 1px solid var(--border-color);
            border-radius: 12px;
            color: #ffffff;
            font-family: inherit;
            font-size: 0.95rem;
            outline: none;
            transition: all 0.3s ease;
        }

        .form-control:focus {
            border-color: var(--accent-purple);
            box-shadow: 0 0 12px rgba(139, 92, 246, 0.15);
            background-color: rgba(255, 255, 255, 0.05);
        }

        .btn-submit {
            width: 100%;
            padding: 14px;
            background: var(--accent-grad);
            border: none;
            border-radius: 12px;
            color: #ffffff;
            font-family: inherit;
            font-size: 1rem;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.3s ease;
            box-shadow: 0 4px 15px rgba(139, 92, 246, 0.25);
            margin-top: 10px;
        }

        .btn-submit:hover {
            transform: translateY(-2px);
            box-shadow: 0 6px 20px rgba(139, 92, 246, 0.45);
        }

        .card-footer {
            text-align: center;
            margin-top: 25px;
            font-size: 0.9rem;
            color: var(--text-secondary);
        }

        .card-footer a {
            color: var(--accent-purple);
            text-decoration: none;
            font-weight: 600;
        }

        .card-footer a:hover {
            text-decoration: underline;
        }

        /* Alert Banners */
        .alert {
            padding: 12px 18px;
            border-radius: 12px;
            font-size: 0.9rem;
            margin-bottom: 20px;
            display: flex;
            align-items: center;
            gap: 10px;
        }

        .alert-danger {
            background-color: rgba(239, 68, 68, 0.1);
            border: 1px solid rgba(239, 68, 68, 0.2);
            color: var(--danger-color);
        }

        .alert-success {
            background-color: rgba(16, 185, 129, 0.1);
            border: 1px solid rgba(16, 185, 129, 0.2);
            color: var(--success-color);
        }

        /* Dashboard specific styles */
        .dashboard-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 30px;
            border-bottom: 1px solid var(--border-color);
            padding-bottom: 20px;
        }

        .stats-container {
            display: flex;
            gap: 20px;
            margin-bottom: 25px;
        }

        .stat-card {
            flex: 1;
            background: rgba(255, 255, 255, 0.02);
            border: 1px solid var(--border-color);
            padding: 15px 20px;
            border-radius: 16px;
            text-align: center;
        }

        .stat-value {
            font-size: 1.8rem;
            font-weight: 700;
            background: var(--accent-grad);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }

        .stat-label {
            font-size: 0.8rem;
            color: var(--text-secondary);
            text-transform: uppercase;
            letter-spacing: 1px;
            margin-top: 4px;
        }

        .todo-input-form {
            display: flex;
            gap: 10px;
            margin-bottom: 25px;
        }

        .todo-input-form .form-control {
            flex-grow: 1;
        }

        .todo-input-form .btn-add {
            background: var(--accent-grad);
            border: none;
            border-radius: 12px;
            color: #ffffff;
            padding: 0 25px;
            font-family: inherit;
            font-weight: 600;
            cursor: pointer;
            box-shadow: 0 4px 12px rgba(139, 92, 246, 0.2);
            transition: all 0.3s ease;
        }

        .todo-input-form .btn-add:hover {
            transform: translateY(-1px);
            box-shadow: 0 6px 18px rgba(139, 92, 246, 0.4);
        }

        .todo-list {
            list-style: none;
            display: flex;
            flex-direction: column;
            gap: 10px;
        }

        .todo-item {
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 16px 20px;
            background-color: rgba(255, 255, 255, 0.02);
            border: 1px solid var(--border-color);
            border-radius: 14px;
            transition: all 0.2s ease;
        }

        .todo-item:hover {
            border-color: rgba(255, 255, 255, 0.15);
            background-color: rgba(255, 255, 255, 0.04);
            transform: translateX(3px);
        }

        .todo-content {
            display: flex;
            align-items: center;
            gap: 12px;
            text-decoration: none;
            color: inherit;
            flex-grow: 1;
            cursor: pointer;
        }

        .checkbox {
            width: 20px;
            height: 20px;
            border-radius: 6px;
            border: 2px solid var(--border-color);
            display: flex;
            align-items: center;
            justify-content: center;
            transition: all 0.2s ease;
        }

        .todo-item.completed .checkbox {
            background-color: var(--success-color);
            border-color: var(--success-color);
        }

        .todo-item.completed .checkbox::after {
            content: "✓";
            color: #ffffff;
            font-size: 0.75rem;
            font-weight: 700;
        }

        .todo-text {
            font-size: 0.95rem;
            font-weight: 500;
            transition: all 0.2s ease;
        }

        .todo-item.completed .todo-text {
            text-decoration: line-through;
            color: var(--text-secondary);
        }

        .btn-delete {
            background: none;
            border: none;
            color: var(--text-secondary);
            cursor: pointer;
            font-size: 1.1rem;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 5px;
            border-radius: 8px;
            transition: all 0.2s ease;
            text-decoration: none;
        }

        .btn-delete:hover {
            color: var(--accent-pink);
            background-color: rgba(236, 72, 153, 0.1);
        }

        .empty-state {
            text-align: center;
            padding: 40px 0;
            color: var(--text-secondary);
        }

        .empty-icon {
            font-size: 2.2rem;
            margin-bottom: 12px;
        }
    </style>
</head>
<body>
    <nav class="navbar">
        <a href="/" class="nav-logo">PHX_CLOUD</a>
        ' . $userSection . '
    </nav>
    <div class="main-content">
        ' . $bodyHtml . '
    </div>
</body>
</html>';

        return $html;
    }
}
?>
