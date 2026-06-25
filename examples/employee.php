<?php

class Employee
{
    private $name;
    private $salary;

    public function __construct($name, $salary)
    {
        $this->name = $name;
        $this->salary = $salary;
    }

    public function display()
    {
        echo "Name: {$this->name}<br>";
        echo "Salary: {$this->salary}";
    }
}

$employee = new Employee("Vicky", 50000);
$employee->display();

?>