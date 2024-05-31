package service

import (
	"testing"
)

func hello(name string) string {
	return "Hello, " + name + "!"
}

func TestHelloWithParameter(t *testing.T) {
	expected := "Hello, Integration-service!"
	actual := hello("Integration-service")
	if actual != expected {
		t.Errorf("Expected %s, but got %s", expected, actual)
	}
}

func TestHelloWithEmptyParameter(t *testing.T) {
	expected := "Hello, World!"
	actual := hello("")
	if actual != expected {
		t.Errorf("Expected %s, but got %s", expected, actual)
	}
}

func TestAdd(t *testing.T) {
	expected := 5
	actual := add(2, 3)
	if actual != expected {
		t.Errorf("Expected %d, but got %d", expected, actual)
	}
}

func TestAddWithNegativeNumbers(t *testing.T) {
	expected := -5
	actual := add(-2, -3)
	if actual != expected {
		t.Errorf("Expected %d, but got %d", expected, actual)
	}
}

func TestAddWithZero(t *testing.T) {
	expected := 10
	actual := add(10, 0)
	if actual != expected {
		t.Errorf("Expected %d, but got %d", expected, actual)
	}
}
