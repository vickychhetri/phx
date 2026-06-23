<?php
echo "========================================\n";
echo "       PHX Interactive Todo CLI        \n";
echo "========================================\n";

$tasks = [];
$completed = [];

while (true) {
    echo "\nMenu Options:\n";
    echo "1. View tasks\n";
    echo "2. Add a new task\n";
    echo "3. Mark task as completed\n";
    echo "4. Delete a task\n";
    echo "5. Exit\n";
    
    $choice = readline("Choose an option: ");
    
    if ($choice == "1") {
        echo "\n--- Current Tasks ---\n";
        $count = count($tasks);
        if ($count == 0) {
            echo "No tasks found.\n";
        } else {
            for ($i = 0; $i < $count; $i++) {
                $status = $completed[$i] ? "[Done]" : "[Pending]";
                echo ($i + 1) . ". " . $status . " " . $tasks[$i] . "\n";
            }
        }
    } else if ($choice == "2") {
        $task = readline("Enter task description: ");
        if ($task == "") {
            echo "Task description cannot be empty.\n";
        } else {
            $tasks[] = $task;
            $completed[] = false;
            echo "Task added successfully.\n";
        }
    } else if ($choice == "3") {
        echo "Enter task number to mark as completed: ";
        $idxStr = readline();
        $idx = intval($idxStr) - 1;
        if ($idx >= 0) {
            if ($idx < count($tasks)) {
                $completed[$idx] = true;
                echo "Task marked as completed!\n";
            } else {
                echo "Invalid task number.\n";
            }
        } else {
            echo "Invalid task number.\n";
        }
    } else if ($choice == "4") {
        echo "Enter task number to delete: ";
        $idxStr = readline();
        $idx = intval($idxStr) - 1;
        if ($idx >= 0) {
            if ($idx < count($tasks)) {
                $newTasks = [];
                $newCompleted = [];
                for ($k = 0; $k < count($tasks); $k++) {
                    if ($k != $idx) {
                        $newTasks[] = $tasks[$k];
                        $newCompleted[] = $completed[$k];
                    }
                }
                $tasks = $newTasks;
                $completed = $newCompleted;
                echo "Task deleted successfully.\n";
            } else {
                echo "Invalid task number.\n";
            }
        } else {
            echo "Invalid task number.\n";
        }
    } else if ($choice == "5") {
        echo "Exiting Todo CLI. Goodbye!\n";
        break;
    } else {
        echo "Invalid choice. Please select 1-5.\n";
    }
}
?>
