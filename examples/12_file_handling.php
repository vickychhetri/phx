<?php
use PHX\File;
use PHX\Exception;

echo "----------------------------------------\n";
echo "         PHX File Handling              \n";
echo "----------------------------------------\n";

$filePath = "temp_demo_file.txt";

try {
    $file = new File();
    
    echo "Opening file: " . $filePath . " for writing...\n";
    $file->open($filePath, "w");
    
    echo "Writing lines to file...\n";
    $file->write("Hello, PHX Standard Library!\n");
    $file->write("Line 2: Built-in packages are working beautifully.\n");
    $file->write("Line 3: File, DB, and Exception packages added.\n");
    $file->close();
    echo "File written and closed.\n\n";
    
    echo "Opening file for reading...\n";
    $file->open($filePath, "r");
    
    echo "Reading file contents:\n";
    $content = $file->read(1000);
    echo $content;
    $file->close();
    
    echo "\nDeleting temporary file...\n";
    $file->delete($filePath);
    echo "File deleted.\n";
    
} catch (Exception $e) {
    echo "File operation failed: " . $e->getMessage() . "\n";
}

echo "----------------------------------------\n";
?>
