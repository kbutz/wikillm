#!/bin/bash

# WikiLLM Test Script
# This script helps test the WikiLLM application with a small Wikipedia dump

set -e  # Exit on error

echo "WikiLLM Test Script"
echo "==================="

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed. Please install Go first."
    exit 1
fi

# Check if Ollama is installed
if ! command -v ollama &> /dev/null; then
    echo "Error: Ollama is not installed. Please install Ollama first."
    echo "Visit: https://ollama.ai/download"
    exit 1
fi

# Check if Ollama is running
if ! curl -s http://localhost:11434/api/tags &> /dev/null; then
    echo "Error: Ollama is not running. Please start Ollama with 'ollama serve'"
    exit 1
fi

echo "Building WikiLLM..."
go build -o wikillm

# Check if we have a test Wikipedia dump
TEST_DUMP="test_wikipedia.xml"
if [ ! -f "$TEST_DUMP" ]; then
    echo "Test Wikipedia dump not found. Creating a small test dump..."
    
    # Create a minimal Wikipedia XML dump for testing
    cat > "$TEST_DUMP" << EOF
<mediawiki>
  <page>
    <title>Albert Einstein</title>
    <id>1</id>
    <revision>
      <text>Albert Einstein was a German-born theoretical physicist who developed the theory of relativity, one of the two pillars of modern physics. His work is also known for its influence on the philosophy of science. He is best known to the general public for his mass–energy equivalence formula E = mc². He received the Nobel Prize in Physics in 1921 "for his services to theoretical physics, and especially for his discovery of the law of the photoelectric effect", a pivotal step in the development of quantum theory.</text>
    </revision>
  </page>
  <page>
    <title>Quantum Mechanics</title>
    <id>2</id>
    <revision>
      <text>Quantum mechanics is a fundamental theory in physics that provides a description of the physical properties of nature at the scale of atoms and subatomic particles. It is the foundation of all quantum physics including quantum chemistry, quantum field theory, quantum technology, and quantum information science.</text>
    </revision>
  </page>
  <page>
    <title>Go (programming language)</title>
    <id>3</id>
    <revision>
      <text>Go is a statically typed, compiled programming language designed at Google by Robert Griesemer, Rob Pike, and Ken Thompson. Go is syntactically similar to C, but with memory safety, garbage collection, structural typing, and CSP-style concurrency.</text>
    </revision>
  </page>
</mediawiki>
EOF
    echo "Created test Wikipedia dump with 3 sample articles."
fi

echo "Running WikiLLM with test data..."
echo "This will create an index and start an interactive session."
echo "Try asking questions like 'Who was Albert Einstein?' or 'What is quantum mechanics?'"
echo ""
echo "Press Ctrl+C to exit the test."
echo ""

# Run WikiLLM with the test dump
./wikillm -wikipedia "$TEST_DUMP" -model llama2