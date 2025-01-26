package dependencies

import (
	"context"
	"testing"
)

func TestPythonDependencyInstallation(t *testing.T) {
	tests := []struct {
		name string
		code string
	}{
		{
			name: "requests library",
			code: `
import requests
response = requests.get('https://httpbin.org/get')
print(response.json())`,
		},
		{
			name: "multiple dependencies",
			code: `
import numpy as np
import pandas as pd
data = pd.DataFrame({'A': np.random.rand(5)})
print(data)`,
		},
		{
			name: "PIL with aliased import",
			code: `
from PIL import Image
import numpy as np
img = Image.new('RGB', (60, 30), color='red')
print(np.array(img).shape)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := ParsePythonImports(tt.code)
			if len(deps) == 0 {
				t.Fatal("No dependencies found")
			}

			// Test that the code runs successfully with dependencies
			output, err := RunWithDependencies(context.Background(), tt.code, Python, deps)
			if err != nil {
				t.Errorf("Failed to run code with dependencies: %v", err)
			}
			if output == "" {
				t.Error("No output received from code execution")
			}
		})
	}
}

func TestNodeDependencyInstallation(t *testing.T) {
	tests := []struct {
		name string
		code string
	}{
		{
			name: "axios library",
			code: `
const axios = require('axios');
axios.get('https://httpbin.org/get')
  .then(response => console.log(response.data))
  .catch(error => console.error(error));`,
		},
		{
			name: "multiple dependencies",
			code: `
import express from 'express';
import cors from 'cors';
const app = express();
app.use(cors());
console.log('Server configured');`,
		},
		{
			name: "scoped package",
			code: `
import { useState } from '@testing-library/react';
console.log(typeof useState);`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := ParseNodeImports(tt.code)
			if len(deps) == 0 {
				t.Fatal("No dependencies found")
			}

			// Test that the code runs successfully with dependencies
			output, err := RunWithDependencies(context.Background(), tt.code, NodeJS, deps)
			if err != nil {
				t.Errorf("Failed to run code with dependencies: %v", err)
			}
			if output == "" {
				t.Error("No output received from code execution")
			}
		})
	}
}

func TestGoDependencyInstallation(t *testing.T) {
	tests := []struct {
		name string
		code string
	}{
		{
			name: "gin framework",
			code: `
package main

import (
    "github.com/gin-gonic/gin"
)

func main() {
    r := gin.New()
    println("Gin router created")
}`,
		},
		{
			name: "multiple dependencies",
			code: `
package main

import (
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)

func main() {
    println("Imports successful")
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := ParseGoImports(tt.code)
			if len(deps) == 0 {
				t.Fatal("No dependencies found")
			}

			// Test that the code runs successfully with dependencies
			output, err := RunWithDependencies(context.Background(), tt.code, Go, deps)
			if err != nil {
				t.Errorf("Failed to run code with dependencies: %v", err)
			}
			if output == "" {
				t.Error("No output received from code execution")
			}
		})
	}
}
