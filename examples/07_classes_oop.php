<?php
// -------------------------------------------------------
// Example 07: Classes & Objects (OOP)
// -------------------------------------------------------

class Animal {
    private $name;
    private $sound;

    public function __construct($name, $sound) {
        $this->name  = $name;
        $this->sound = $sound;
    }

    public function speak() {
        echo $this->name . " says: " . $this->sound . "\n";
    }

    public function getName() {
        return $this->name;
    }
}

// Instantiate objects
$cat  = new Animal("Whiskers", "Meow");
$dog  = new Animal("Rex", "Woof");
$bird = new Animal("Tweety", "Tweet");

$cat->speak();
$dog->speak();
$bird->speak();

echo "Cat's name: " . $cat->getName() . "\n";

// -------------------------------------------------------
// Bank Account class
// -------------------------------------------------------
class BankAccount {
    private $owner;
    private $balance;

    public function __construct($owner, $balance) {
        $this->owner   = $owner;
        $this->balance = $balance;
    }

    public function deposit($amount) {
        $this->balance += $amount;
        echo "Deposited " . $amount . " | Balance: " . $this->balance . "\n";
    }

    public function withdraw($amount) {
        if ($amount > $this->balance) {
            echo "Insufficient funds!\n";
        } else {
            $this->balance -= $amount;
            echo "Withdrew " . $amount . " | Balance: " . $this->balance . "\n";
        }
    }

    public function getBalance() {
        return $this->balance;
    }
}

echo "\n--- Bank Account ---\n";
$account = new BankAccount("Vicky", 1000);
$account->deposit(500);
$account->withdraw(300);
$account->withdraw(2000);
echo "Final Balance: " . $account->getBalance() . "\n";

// -------------------------------------------------------
// Stack class
// -------------------------------------------------------
class Stack {
    private $data;
    private $size;

    public function __construct() {
        $this->data = [];
        $this->size = 0;
    }

    public function push($val) {
        $this->data[] = $val;
        $this->size++;
    }

    public function pop() {
        if ($this->size == 0) {
            return null;
        }
        $this->size--;
        $top = $this->data[$this->size];
        return $top;
    }

    public function peek() {
        if ($this->size == 0) {
            return null;
        }
        return $this->data[$this->size - 1];
    }

    public function isEmpty() {
        return $this->size == 0;
    }

    public function length() {
        return $this->size;
    }
}

echo "\n--- Stack ---\n";
$stack = new Stack();
$stack->push(10);
$stack->push(20);
$stack->push(30);
echo "Peek   : " . $stack->peek() . "\n";
echo "Pop    : " . $stack->pop() . "\n";
echo "Pop    : " . $stack->pop() . "\n";
echo "Length : " . $stack->length() . "\n";
?>
