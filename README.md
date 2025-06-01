# wikillm
This was supposed to be a repo for getting a version of an offline "internet in a box" with an LLM + Wikipedia set up, 
but am using it anything related to me learning/messing around with LLM patterns instead now.

## Sub-modules

### agents
A playground for experimenting with interactive tools using LLM agents. Provides a command-line interface for managing a to-do list with natural language commands.
The focus of this module is to experiment with the tool -> llm interaction pattern, where the LLM can call tools to perform actions based on user input.

### inmemory
Runs an LLM locally with an offline version of Wikipedia loaded into memory. Allows querying and interacting with Wikipedia data without an internet connection.

### qdrant
A Retrieval-Augmented Generation (RAG) system for querying Wikipedia content using Qdrant vector database. Supports multiple LLM providers and advanced embedding options.

### tool
An enhanced version of the to-do list agent with both command-line and HTTP interfaces. Provides advanced analytical capabilities for task management.
This used a deprecated `/v1/completions` endpoint, so it has been updated to use the `/v1/chat/completions` endpoint in the `agents` module.
