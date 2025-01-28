package languages

import (
	"testing"
)

func TestParsePythonImports(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected []string
	}{
		{
			name: "simple imports",
			code: `
import requests
import pandas as pd
from PIL import Image`,
			expected: []string{"requests", "pandas", "pillow"},
		},
		{
			name: "standard library only",
			code: `
import os
import sys
from datetime import datetime`,
			expected: []string{},
		},
		{
			name: "mixed imports",
			code: `
import os
import requests
from datetime import datetime
import numpy as np
from tensorflow import keras`,
			expected: []string{"requests", "numpy", "tensorflow"},
		},
		{
			name: "multiline imports",
			code: `
from fastapi import (
    FastAPI,
    HTTPException,
    Depends,
)
import numpy as np`,
			expected: []string{"fastapi", "numpy"},
		},
		{
			name: "commented imports",
			code: `
# import requests
import numpy as np
# from PIL import Image`,
			expected: []string{"numpy"},
		},
		{
			name: "string literals with import keyword",
			code: `
x = "import requests"
import numpy as np`,
			expected: []string{"numpy"},
		},
		{
			name: "dynamic imports",
			code: `
np = __import__('numpy')
requests = __import__('requests')`,
			expected: []string{"numpy", "requests"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParsePythonImports(tt.code)
			if !equalStringSlices(got, tt.expected) {
				t.Errorf("ParsePythonImports() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseNodeImports(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected []string
	}{
		{
			name: "require statements",
			code: `
const express = require('express');
const axios = require('axios');`,
			expected: []string{"express", "axios"},
		},
		{
			name: "ES6 imports",
			code: `
import axios from 'axios';
import { useState } from 'react';
import * as d3 from 'd3';`,
			expected: []string{"axios", "react", "d3"},
		},
		{
			name: "built-in modules only",
			code: `
const fs = require('fs');
const path = require('path');
import { Buffer } from 'buffer';`,
			expected: []string{},
		},
		{
			name: "mixed imports",
			code: `
const fs = require('fs');
const express = require('express');
import { useState } from 'react';
const path = require('path');`,
			expected: []string{"express", "react"},
		},
		{
			name: "commented imports",
			code: `
// const express = require('express');
const axios = require('axios');
// import { useState } from 'react';`,
			expected: []string{"axios"},
		},
		{
			name: "string literals with require",
			code: `
const x = "const express = require('express')";
const axios = require('axios');`,
			expected: []string{"axios"},
		},
		{
			name: "dynamic imports",
			code: `
const mod = await import('lodash');
import('react').then(React => {});`,
			expected: []string{"lodash", "react"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseNodeImports(tt.code)
			if !equalStringSlices(got, tt.expected) {
				t.Errorf("ParseNodeImports() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseGoImports(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected []string
	}{
		{
			name: "single line imports",
			code: `
package main

import "github.com/gin-gonic/gin"
import "gorm.io/gorm"`,
			expected: []string{"github.com/gin-gonic/gin", "gorm.io/gorm"},
		},
		{
			name: "grouped imports",
			code: `
package main

import (
    "fmt"
    "github.com/gin-gonic/gin"
    "os"
    "gorm.io/gorm"
)`,
			expected: []string{"github.com/gin-gonic/gin", "gorm.io/gorm"},
		},
		{
			name: "standard library only",
			code: `
package main

import (
    "fmt"
    "os"
    "strings"
)`,
			expected: []string{},
		},
		{
			name: "commented imports",
			code: `
package main

import (
    // "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)`,
			expected: []string{"gorm.io/gorm"},
		},
		{
			name: "named imports",
			code: `
package main

import (
    gin "github.com/gin-gonic/gin"
    db "gorm.io/gorm"
)`,
			expected: []string{"github.com/gin-gonic/gin", "gorm.io/gorm"},
		},
		{
			name: "dot imports",
			code: `
package main

import (
    . "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)`,
			expected: []string{"github.com/gin-gonic/gin", "gorm.io/gorm"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseGoImports(tt.code)
			if !equalStringSlices(got, tt.expected) {
				t.Errorf("ParseGoImports() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// Helper function to compare string slices regardless of order
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[string]int)
	for _, s := range a {
		seen[s]++
	}
	for _, s := range b {
		seen[s]--
		if seen[s] < 0 {
			return false
		}
	}
	return true
}
