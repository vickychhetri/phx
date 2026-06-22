package vm

import (
	"bytes"
	"phx/internal/lexer"
	"phx/internal/parser"
	"testing"
)

func TestEvaluateProgram(t *testing.T) {
	tests := []struct {
		input          string
		expectedOutput string
		expectedEnv    map[string]interface{}
	}{
		{
			input:          `<html>Hello World</html>`,
			expectedOutput: `<html>Hello World</html>`,
			expectedEnv:    map[string]interface{}{},
		},
		{
			input: `<?php
$name = "Vicky";
$greeting = "Hello, " . $name;
echo $greeting, "\n";
?>`,
			expectedOutput: "Hello, Vicky\n",
			expectedEnv: map[string]interface{}{
				"$name":     "Vicky",
				"$greeting": "Hello, Vicky",
			},
		},
		{
			input: `<?php
$x = 10;
$y = 3.5;
$sum = $x + $y;
if ($sum > 13) {
	echo "Greater";
} else {
	echo "Smaller";
}
?>`,
			expectedOutput: "Greater",
			expectedEnv: map[string]interface{}{
				"$x":   int64(10),
				"$y":   3.5,
				"$sum": 13.5,
			},
		},
		{
			input: `<?php
$val = true;
if ($val === true) {
	echo "Yes";
} else {
	echo "No";
}
?>`,
			expectedOutput: "Yes",
			expectedEnv: map[string]interface{}{
				"$val": true,
			},
		},
		{
			input: `<?php
function add($a, $b) {
	return $a + $b;
}
echo add(5, 7);
?>`,
			expectedOutput: "12",
			expectedEnv:    map[string]interface{}{},
		},
		{
			input: `<?php
class Calculator {
	public function add($a, $b) {
		return $a + $b;
	}
}
$calc = new Calculator();
echo $calc->add(10, 20);
?>`,
			expectedOutput: "30",
			expectedEnv:    map[string]interface{}{},
		},
		{
			input: `<?php
$op = "+";
$res = 0;
switch ($op) {
	case "+":
		$res = 5 + 5;
		break;
	case "-":
		$res = 5 - 5;
		break;
}
echo $res;
?>`,
			expectedOutput: "10",
			expectedEnv: map[string]interface{}{
				"$res": int64(10),
			},
		},
		{
			input: `<?php
$val = (5 > 3) ? "Yes" : "No";
echo $val;
?>`,
			expectedOutput: "Yes",
			expectedEnv: map[string]interface{}{
				"$val": "Yes",
			},
		},
		{
			input: `<?php
function greet($name = "World") {
	return "Hello " . $name;
}
echo greet();
echo " ";
echo greet("Vicky");
?>`,
			expectedOutput: "Hello World Hello Vicky",
			expectedEnv:    map[string]interface{}{},
		},
		{
			input: `<?php
class BankAccount {
	private $balance;
	public function __construct($balance = 0) {
		$this->balance = $balance;
	}
	public function deposit($amount) {
		$this->balance += $amount;
	}
	public function getBalance() {
		return $this->balance;
	}
}
$acc = new BankAccount(100);
$acc->deposit(50);
echo $acc->getBalance();
?>`,
			expectedOutput: "150",
			expectedEnv:    map[string]interface{}{},
		},
		{
			input: `<?php
$val = -5;
$notVal = !true;
echo $val;
echo " ";
echo $notVal ? "true" : "false";
?>`,
			expectedOutput: "-5 false",
			expectedEnv: map[string]interface{}{
				"$val":    int64(-5),
				"$notVal": false,
			},
		},
		{
			input: `<?php
$name = "Vicky";
$salary = 50000;
echo "Name: {$name}, Salary: {$salary}";
?>`,
			expectedOutput: "Name: Vicky, Salary: 50000",
			expectedEnv: map[string]interface{}{
				"$name":   "Vicky",
				"$salary": int64(50000),
			},
		},
		{
			input: `<?php
$i = 1;
while ($i <= 3) {
	echo $i;
	$i = $i + 1;
}
?>`,
			expectedOutput: "123",
			expectedEnv: map[string]interface{}{
				"$i": int64(4),
			},
		},
		{
			input: `<?php
for ($j = 1; $j <= 5; $j = $j + 1) {
	if ($j === 2) { continue; }
	if ($j === 4) { break; }
	echo $j;
}
?>`,
			expectedOutput: "13",
			expectedEnv: map[string]interface{}{
				"$j": int64(4),
			},
		},
		{
			input: `<?php
$i = 5;
echo $i++;
echo " ";
echo $i;
?>`,
			expectedOutput: "5 6",
			expectedEnv: map[string]interface{}{
				"$i": int64(6),
			},
		},
		{
			input: `<?php
$i = 5;
echo ++$i;
echo " ";
echo $i;
?>`,
			expectedOutput: "6 6",
			expectedEnv: map[string]interface{}{
				"$i": int64(6),
			},
		},
		{
			input: `<?php
namespace App;
class Hello {
	public function say() {
		return "Hello Namespace!";
	}
}

namespace Runner;
use App\Hello;
$h = new Hello();
echo $h->say();
?>`,
			expectedOutput: "Hello Namespace!",
			expectedEnv: map[string]interface{}{},
		},
		{
			input: `<?php
$ch = new Channel(1);
$t = spawn(function($c) {
	$c->send("Hello from Thread");
}, $ch);
echo $ch->recv();
$t->join();
?>`,
			expectedOutput: "Hello from Thread",
			expectedEnv: map[string]interface{}{},
		},
		{
			input: `<?php
$arr = [10, 20];
$arr[] = 30;
$hash = ["x" => 100];
echo $arr[2] + $hash["x"] + strlen("abc") + max(1, 5);
?>`,
			expectedOutput: "138",
			expectedEnv: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := parser.New(l)
		program := p.ParseProgram()
		if len(p.Errors()) > 0 {
			t.Fatalf("parser errors: %v", p.Errors())
		}

		env := NewEnvironment()
		var out bytes.Buffer
		res := Evaluate(program, env, &out)

		if isError(res) {
			t.Fatalf("eval error: %s", res.(*Error).Message)
		}

		if out.String() != tt.expectedOutput {
			t.Errorf("expected output %q, got %q", tt.expectedOutput, out.String())
		}

		for varName, expectedVal := range tt.expectedEnv {
			val, ok := env.Get(varName)
			if !ok {
				t.Errorf("variable %s not found in environment", varName)
				continue
			}

			switch ev := expectedVal.(type) {
			case string:
				if val.Type() != STRING_OBJ || val.(*String).Value != ev {
					t.Errorf("expected %s to be String(%q), got %s(%q)", varName, ev, val.Type(), val.Inspect())
				}
			case int64:
				if val.Type() != INTEGER_OBJ || val.(*Integer).Value != ev {
					t.Errorf("expected %s to be Integer(%d), got %s(%q)", varName, ev, val.Type(), val.Inspect())
				}
			case float64:
				if val.Type() != FLOAT_OBJ || val.(*Float).Value != ev {
					t.Errorf("expected %s to be Float(%g), got %s(%q)", varName, ev, val.Type(), val.Inspect())
				}
			case bool:
				if val.Type() != BOOLEAN_OBJ || val.(*Boolean).Value != ev {
					t.Errorf("expected %s to be Boolean(%t), got %s(%q)", varName, ev, val.Type(), val.Inspect())
				}
			}
		}
	}
}
