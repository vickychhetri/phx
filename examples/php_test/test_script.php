<?php

class BankAccount
{
    private $balance;

    public function __construct($balance = 0)
    {
        $this->balance = $balance;
    }

    public function deposit($amount)
    {
        $this->balance += $amount;
    }

    public function withdraw($amount)
    {
        if ($amount <= $this->balance) {
            $this->balance -= $amount;
        } else {
            echo "Insufficient Balance<br>";
        }
    }

    public function getBalance()
    {
        return $this->balance;
    }
}

$account = new BankAccount(1000);

$account->deposit(900);
$account->withdraw(400);

echo "Current Balance: " . $account->getBalance();

?>