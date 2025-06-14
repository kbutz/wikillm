# Enhanced Memory System for WikiLLM Agents

## Overview

The enhanced memory system provides persistent, context-aware memory capabilities for your AI agent. Unlike the basic file memory tool, this system automatically detects, categorizes, and retrieves relevant information without explicit user commands.

## Key Features

### 1. **Automatic Memory Detection**
The agent automatically stores information when it detects:
- Project mentions ("working on", "project called")
- User preferences ("I prefer", "always use")
- Tasks and deadlines ("need to", "by tomorrow")
- Technical decisions ("decided to use", "going with")
- Personal information ("my role", "I work at")

### 2. **Categorized Storage**
Memories are organized into categories:
- `user_profile`: Personal information and preferences
- `projects`: Project details and status
- `tasks`: Tasks with deadlines and priorities
- `technical_details`: Technical stack and architecture decisions
- `decisions`: Important decisions and their rationale
- `conversations`: Significant exchanges

### 3. **Intelligent Retrieval**
The agent automatically:
- Loads relevant context at startup
- Searches for related memories before responding
- Updates its responses based on stored context

### 4. **Structured Metadata**
Each memory includes:
- Timestamps (created, accessed, modified)
- Tags for easy filtering
- Custom metadata (deadlines, priorities, status)

## Usage

### Running with Enhanced Memory (Default)

```bash
./agents --enhanced-memory=true --memory-dir=./memory
```

### Command Line Options

- `--enhanced-memory`: Enable enhanced memory system (default: true)
- `--memory-dir`: Directory for memory storage (default: "memory")
- `--model`: LLM model name
- `--provider`: Model provider (lmstudio or ollama)
- `--debug`: Enable debug logging

### Memory Tool Commands

While the system works automatically, you can also use explicit commands:

```
# Store a memory
> Use the enhanced_memory tool to store projects "Working on WikiLLM v2.0" with tags: development, ai

# Retrieve memories
> Use the enhanced_memory tool to retrieve projects

# Search memories  
> Use the enhanced_memory tool to search "WikiLLM"

# Get context summary
> Use the enhanced_memory tool to show context
```

## Examples

### Example 1: Project Memory

```
> I'm working on a new project called DataAnalyzer using Python and pandas
Processing your request...

[Agent automatically stores this as a project memory]

Response:
I've noted that you're working on DataAnalyzer using Python and pandas. This project information has been stored in my memory. 

What aspects of the DataAnalyzer project would you like to discuss? I can help with:
- Setting up the project structure
- Creating data processing pipelines with pandas
- Suggesting best practices for Python data analysis projects

Response generated in 0.82 seconds.
```

### Example 2: Preference Memory

```
> I prefer using type hints in all my Python code and following PEP 8 strictly
Processing your request...

[Agent automatically stores this preference]

Response:
I've stored your coding preferences. Going forward, I'll ensure that:
- All Python code examples include comprehensive type hints
- Code strictly follows PEP 8 style guidelines

These preferences will be applied to all future Python code I generate for you.

Response generated in 0.65 seconds.
```

### Example 3: Context-Aware Responses

```
> How should I structure the DataAnalyzer project?
Processing your request...

[Agent searches memory and finds the project details and user preferences]

Response:
Based on your DataAnalyzer project using Python and pandas, and your preference for type hints and PEP 8 compliance, here's a recommended structure:

```
DataAnalyzer/
├── src/
│   ├── __init__.py
│   ├── data_loader.py      # Type-hinted data loading functions
│   ├── processors.py       # Data transformation with pandas
│   ├── analyzers.py        # Analysis functions
│   └── utils.py           # Helper functions
├── tests/
│   ├── test_data_loader.py
│   ├── test_processors.py
│   └── test_analyzers.py
├── data/
│   ├── raw/              # Original data files
│   └── processed/        # Cleaned data
├── notebooks/            # Jupyter notebooks for exploration
├── requirements.txt
├── setup.py
├── .flake8              # PEP 8 configuration
└── mypy.ini            # Type checking configuration
```

All modules will include proper type hints as you prefer.

Response generated in 1.23 seconds.
```

### Example 4: Task Management with Memory

```
> I need to finish the data validation module by 2024-06-20
Processing your request...

[Agent stores this as a task with deadline]

Response:
I've added this task to your memory:
- Task: Finish data validation module
- Deadline: 2024-06-20
- Project: DataAnalyzer (from context)

Would you like me to help you break down the data validation module into smaller subtasks or create a implementation plan?

Response generated in 0.71 seconds.
```

## How It Works

### Memory Storage

Memories are stored as JSON files in the specified directory:
```
memory/
├── index.json                              # Memory index for fast retrieval
├── projects_20240614_143022.json          # Individual memory entries
├── user_profile_20240614_143045.json
└── tasks_20240614_143108.json
```

### Memory Lifecycle

1. **Detection**: Agent analyzes user input for memory triggers
2. **Categorization**: Determines appropriate category and extracts metadata
3. **Storage**: Saves memory with timestamps and indexing
4. **Retrieval**: Searches relevant memories for context
5. **Application**: Uses memories to enhance responses

### Integration with LLM

The system modifies the LLM's system prompt to include:
- Instructions for proactive memory usage
- Guidelines for what to remember
- Context from previous conversations

## Advanced Features

### Memory Search

The agent performs intelligent searches across all memories:
- Content-based search
- Category filtering
- Tag-based retrieval
- Temporal ordering (most recent first)

### Memory Updates

When new information contradicts stored memories, the agent:
- Updates the existing memory
- Maintains modification history
- Preserves the original creation timestamp

### Context Initialization

On startup, the agent:
- Loads user profile information
- Retrieves active projects
- Shows recent tasks
- Summarizes recent decisions

## Best Practices

1. **Let the agent work automatically** - Don't explicitly tell it to remember things
2. **Be specific** - Clear project names and deadlines help categorization
3. **Update regularly** - Mention status changes for the agent to track
4. **Review periodically** - Use "show context" to see what's stored

## Troubleshooting

### Memory Not Being Stored

If memories aren't being stored:
1. Check that `--enhanced-memory=true` is set
2. Verify write permissions for the memory directory
3. Enable debug mode to see memory operations
4. Ensure your statements include trigger keywords

### Context Not Loading

If context isn't loading at startup:
1. Check that memory files exist in the directory
2. Verify the index.json file is valid
3. Look for initialization errors in debug mode

### Performance Issues

For large memory stores:
1. Periodically clean old memories
2. Use specific searches rather than browsing all
3. Consider archiving completed project memories

## Privacy and Security

- All memories are stored locally in your specified directory
- No data is sent to external services
- You can manually edit or delete memory files
- Consider encrypting the memory directory for sensitive data

## Future Enhancements

Planned improvements include:
- Semantic search using embeddings
- Memory compression for older entries
- Automatic memory pruning policies
- Export/import functionality
- Memory sharing between agents
