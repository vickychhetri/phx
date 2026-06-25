<?php
// views/dashboard.php

class DashboardView {
    public function render($todos, $username) {
        $total = count($todos);
        $completed = 0;
        for ($i = 0; $i < $total; $i++) {
            $t = $todos[$i];
            // Normalize MySQL boolean result which might be string "1"/"0" or bool true/false
            $isCompleted = false;
            if ($t["completed"] === true) {
                $isCompleted = true;
            }
            if ($t["completed"] === 1) {
                $isCompleted = true;
            }
            if ($t["completed"] === "1") {
                $isCompleted = true;
            }
            if ($t["completed"] === "true") {
                $isCompleted = true;
            }
            
            if ($isCompleted) {
                $completed = $completed + 1;
            }
        }
        
        $percentage = 0;
        if ($total > 0) {
            $percentage = ($completed * 100) / $total;
        }
        
        $listHtml = "";
        if ($total === 0) {
            $listHtml = '<div class="empty-state">
                <div class="empty-icon">🎉</div>
                <p>No tasks found. Add a new task below to get started!</p>
            </div>';
        } else {
            $listHtml = '<ul class="todo-list">';
            for ($i = 0; $i < $total; $i++) {
                $t = $todos[$i];
                $id = $t["id"];
                $title = $t["title"];
                
                $isCompleted = false;
                if ($t["completed"] === true) {
                    $isCompleted = true;
                }
                if ($t["completed"] === 1) {
                    $isCompleted = true;
                }
                if ($t["completed"] === "1") {
                    $isCompleted = true;
                }
                if ($t["completed"] === "true") {
                    $isCompleted = true;
                }
                
                $itemClass = "todo-item";
                if ($isCompleted) {
                    $itemClass = "todo-item completed";
                }
                
                $listHtml = $listHtml . '<li class="' . $itemClass . '">
                    <a href="/todos/toggle/' . $id . '" class="todo-content">
                        <div class="checkbox"></div>
                        <span class="todo-text">' . $title . '</span>
                    </a>
                    <a href="/todos/delete/' . $id . '" class="btn-delete">✕</a>
                </li>';
            }
            $listHtml = $listHtml . '</ul>';
        }
        
        $bodyHtml = '<div class="card dashboard-card">
            <div class="dashboard-header">
                <div>
                    <h1 class="card-title">Task Pipeline</h1>
                    <p class="card-desc">Concurrently managed via PHX Runtime & MySQL</p>
                </div>
            </div>
            
            <div class="stats-container">
                <div class="stat-card">
                    <div class="stat-value">' . $total . '</div>
                    <div class="stat-label">Total Tasks</div>
                </div>
                <div class="stat-card">
                    <div class="stat-value">' . $completed . '</div>
                    <div class="stat-label">Completed</div>
                </div>
                <div class="stat-card">
                    <div class="stat-value">' . number_format($percentage, 0) . '%</div>
                    <div class="stat-label">Progress</div>
                </div>
            </div>
            
            <form class="todo-input-form" action="/todos/add" method="POST">
                <input class="form-control" type="text" name="title" placeholder="Allocate new task task..." required autocomplete="off">
                <button type="submit" class="btn-add">Add</button>
            </form>
            
            ' . $listHtml . '
        </div>';
        
        $layout = new Layout();
        return $layout->render("Dashboard", $bodyHtml, $username);
    }
}
?>
