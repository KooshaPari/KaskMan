package vibes

import (
	"testing"

	"github.com/kooshapari/kodevibe-go/internal/models"
)

func TestCodeChecker_hasTodoComment(t *testing.T) {
	checker := NewCodeChecker()
	
	tests := []struct {
		name     string
		line     string
		expected bool
	}{
		{"TODO comment", "// TODO: Fix this later", true},
		{"FIXME comment", "# FIXME: This is broken", true},
		{"HACK comment", "/* HACK: Quick fix */", true},
		{"XXX comment", "* XXX: Review this", true},
		{"BUG comment", "// BUG: This doesn't work", true},
		{"Regular comment", "// This is a normal comment", false},
		{"No comment", "var x = 5;", false},
		{"Case insensitive", "// todo: lowercase", true},
		{"Mixed case", "// ToDo: mixed case", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.hasTodoComment(tt.line)
			if result != tt.expected {
				t.Errorf("hasTodoComment(%q) = %v, expected %v", tt.line, result, tt.expected)
			}
		})
	}
}

func TestCodeChecker_hasCommentedCode(t *testing.T) {
	checker := NewCodeChecker()
	
	tests := []struct {
		name     string
		line     string
		expected bool
	}{
		{"Commented JavaScript var", "// var x = 5;", true},
		{"Commented JavaScript function", "// function test() {", true},
		{"Commented Python def", "# def test_function():", true},
		{"Commented Python import", "# import os", true},
		{"Regular comment", "// This is a comment", false},
		{"No comment", "var x = 5;", false},
		{"Commented block", "/* function test() { */", true},
		{"Python regular comment", "# This is a normal comment", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.hasCommentedCode(tt.line)
			if result != tt.expected {
				t.Errorf("hasCommentedCode(%q) = %v, expected %v", tt.line, result, tt.expected)
			}
		})
	}
}

func TestCodeChecker_hasMagicNumber(t *testing.T) {
	checker := NewCodeChecker()
	
	tests := []struct {
		name     string
		line     string
		expected bool
	}{
		{"Magic number", "timeout = 12345;", true},
		{"Common number", "count = 1;", false},
		{"Port number", "port = 8080;", false},
		{"Version number", "version = 1.2.3;", false},
		{"Array index", "arr[0] = value;", false},
		{"Timeout assignment", "timeout = 5000;", false},
		{"Size assignment", "size = 1024;", false},
		{"Comment with number", "// Set to 12345", false},
		{"Hex number", "color = 0xFF0000;", false},
		{"Decimal number", "pi = 3.14159;", false},
		{"For loop", "for (i = 0; i < 100; i++)", false},
		{"Sleep call", "sleep(1000);", false},
		{"Buffer size", "buffer = 2048;", false},
		{"True magic number", "threshold = 42137;", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.hasMagicNumber(tt.line)
			if result != tt.expected {
				t.Errorf("hasMagicNumber(%q) = %v, expected %v", tt.line, result, tt.expected)
			}
		})
	}
}

func TestCodeChecker_isCommonNumber(t *testing.T) {
	checker := NewCodeChecker()
	
	tests := []struct {
		name     string
		number   string
		expected bool
	}{
		{"Zero", "0", true},
		{"One", "1", true},
		{"Common port", "8080", true},
		{"Power of 2", "1024", true},
		{"Uncommon number", "12345", false},
		{"Time constant", "86400", true},
		{"Network limit", "65535", true},
		{"Small prime", "17", true},
		{"Large uncommon", "999999", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.isCommonNumber(tt.number)
			if result != tt.expected {
				t.Errorf("isCommonNumber(%q) = %v, expected %v", tt.number, result, tt.expected)
			}
		})
	}
}

func TestCodeChecker_isFunctionStart(t *testing.T) {
	checker := NewCodeChecker()
	
	tests := []struct {
		name     string
		line     string
		ext      string
		expected bool
	}{
		// JavaScript tests
		{"JS function", "function test() {", ".js", true},
		{"JS const arrow", "const test = () => {", ".js", true},
		{"JS let arrow", "let test = () => {", ".js", true},
		{"JS async function", "async function test() {", ".js", true},
		{"JS export function", "export function test() {", ".js", true},
		{"JS method", "test() {", ".js", true},
		{"JS object method", "test: function() {", ".js", true},
		{"JS not function", "const x = 5;", ".js", false},
		
		// Python tests
		{"Python function", "def test():", ".py", true},
		{"Python async function", "async def test():", ".py", true},
		{"Python decorator", "@decorator", ".py", true},
		{"Python not function", "x = 5", ".py", false},
		
		// Go tests
		{"Go function", "func test() {", ".go", true},
		{"Go method", "func (r *Receiver) test() {", ".go", true},
		{"Go not function", "var x = 5", ".go", false},
		
		// Java tests
		{"Java method", "public void test() {", ".java", true},
		{"Java private method", "private int test() {", ".java", true},
		{"Java annotation", "@Override", ".java", true},
		{"Java not method", "int x = 5;", ".java", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.isFunctionStart(tt.line, tt.ext)
			if result != tt.expected {
				t.Errorf("isFunctionStart(%q, %q) = %v, expected %v", tt.line, tt.ext, result, tt.expected)
			}
		})
	}
}

func TestCodeChecker_checkJavaScript(t *testing.T) {
	checker := NewCodeChecker()
	
	tests := []struct {
		name          string
		line          string
		expectedCount int
		expectedRule  string
	}{
		{"var usage", "var x = 5;", 1, "no-var"},
		{"let usage", "let x = 5;", 0, ""},
		{"console.log", "console.log('test');", 1, "no-console-log"},
		{"loose equality", "if (x == 5) {", 1, "strict-equality"},
		{"strict equality", "if (x === 5) {", 0, ""},
		{"not equal", "if (x !== 5) {", 0, ""},
		{"assignment", "x = 5;", 0, ""},
		{"comparison in string", "text = 'x == 5';", 0, ""}, // Should not trigger
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := checker.checkJavaScript("test.js", tt.line, 1)
			if len(issues) != tt.expectedCount {
				t.Errorf("checkJavaScript(%q) returned %d issues, expected %d", tt.line, len(issues), tt.expectedCount)
			}
			if tt.expectedCount > 0 && len(issues) > 0 {
				if issues[0].Rule != tt.expectedRule {
					t.Errorf("checkJavaScript(%q) returned rule %q, expected %q", tt.line, issues[0].Rule, tt.expectedRule)
				}
			}
		})
	}
}

func TestCodeChecker_Configure(t *testing.T) {
	checker := NewCodeChecker()
	
	config := models.VibeConfig{
		Enabled: true,
		Settings: map[string]interface{}{
			"max_function_length":   100,
			"max_nesting_depth":     5,
			"max_line_length":       80,
			"complexity_threshold":  15,
		},
	}
	
	err := checker.Configure(config)
	if err != nil {
		t.Errorf("Configure() returned error: %v", err)
	}
	
	if checker.maxFunctionLength != 100 {
		t.Errorf("maxFunctionLength = %d, expected 100", checker.maxFunctionLength)
	}
	if checker.maxNestingDepth != 5 {
		t.Errorf("maxNestingDepth = %d, expected 5", checker.maxNestingDepth)
	}
	if checker.maxLineLength != 80 {
		t.Errorf("maxLineLength = %d, expected 80", checker.maxLineLength)
	}
	if checker.complexityThreshold != 15 {
		t.Errorf("complexityThreshold = %d, expected 15", checker.complexityThreshold)
	}
}

func TestCodeChecker_Supports(t *testing.T) {
	checker := NewCodeChecker()
	
	tests := []struct {
		filename string
		expected bool
	}{
		{"test.js", true},
		{"test.jsx", true},
		{"test.ts", true},
		{"test.tsx", true},
		{"test.py", true},
		{"test.go", true},
		{"test.java", true},
		{"test.cpp", true},
		{"test.c", true},
		{"test.h", true},
		{"test.hpp", true},
		{"test.cs", true},
		{"test.php", true},
		{"test.rb", true},
		{"test.rs", true},
		{"test.swift", true},
		{"test.kt", true},
		{"test.txt", false},
		{"test.md", false},
		{"test.xml", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := checker.Supports(tt.filename)
			if result != tt.expected {
				t.Errorf("Supports(%q) = %v, expected %v", tt.filename, result, tt.expected)
			}
		})
	}
}