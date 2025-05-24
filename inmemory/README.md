# WikiLLM

WikiLLM is a Go application that runs an LLM locally, loads an offline version of Wikipedia into that LLM, and allows you to make requests to the LLM to interact with and ask questions about that data.

## Features

- Run LLMs locally without an internet connection
- Load and index Wikipedia data for offline access
- Interactive query interface
- Ability to swap between different LLM models and providers (LM Studio and Ollama)
- Fully offline operation

## Prerequisites

Before you begin, ensure you have the following installed:

1. **Go** (version 1.18 or later) - [Download Go](https://golang.org/dl/)
2. **LM Studio** - [Install LM Studio](https://lmstudio.ai/) (if you plan to use LM Studio as your provider)
   - LM Studio is used to run LLM models locally
3. **Ollama** - [Install Ollama](https://ollama.ai/download) (if you plan to use Ollama as your provider)
   - Ollama is an alternative for running LLM models locally

## Installation

1. Clone the repository:

```bash
git clone https://github.com/yourusername/wikillm.git
cd wikillm/inmemory
```

2. Build the application:

```bash
go build
```

## Getting Wikipedia Data

You'll need a Wikipedia dump file to create the index. You can download one from:
https://dumps.wikimedia.org/

For testing, you might want to start with a smaller Wikipedia dump, such as Simple English Wikipedia:
https://dumps.wikimedia.org/simplewiki/latest/simplewiki-latest-pages-articles.xml.bz2

After downloading, extract the bz2 file:

```bash
bunzip2 simplewiki-latest-pages-articles.xml.bz2
```

## Usage

### First Run (Creating the Index)

The first time you run the application, you need to create an index from the Wikipedia dump:

```bash
./inmemory -wikipedia /path/to/wikipedia-dump.xml
```

This will create an index in the default location (`./wikipedia_index`). This process may take some time depending on the size of the Wikipedia dump.

### Regular Usage

Once the index is created, you can run the application without the `-wikipedia` flag:

```bash
./inmemory
```

### Command Line Options

- `-model <model_name>`: Specify the LLM model to use (default: "default")
- `-provider <provider>`: Specify the model provider to use (default: "lmstudio", options: "lmstudio" or "ollama")
- `-wikipedia <path>`: Path to the Wikipedia dump file (only needed for initial indexing)
- `-index <path>`: Directory to store the search index (default: "./wikipedia_index")
- `-limit <number>`: Maximum number of search results to return (default: 5)

Example:

```bash
./inmemory -model llama3 -provider lmstudio -index /data/wiki_index -limit 10
```

## Interactive Session

Once the application is running, you'll enter an interactive session where you can ask questions:

```
WikiLLM Interactive Session
Using model: default
Type 'exit' to quit
> Who was Albert Einstein?
Searching Wikipedia and generating response...

Response (generated in 2.75 seconds):
Albert Einstein was a German-born theoretical physicist who is widely regarded as one of the greatest and most influential physicists of all time. He developed the theory of relativity, one of the two pillars of modern physics (alongside quantum mechanics). His work is also known for its influence on the philosophy of science.

Born in Ulm, Germany in 1879, Einstein is best known for developing the theory of relativity, particularly the mass-energy equivalence formula E = mc², which has been dubbed "the world's most famous equation". He also made important contributions to the development of quantum mechanics, statistical mechanics, and cosmology. For his explanation of the photoelectric effect, he received the Nobel Prize in Physics in 1921.

Einstein published more than 300 scientific papers and more than 150 non-scientific works. His intellectual achievements and originality have made the word "Einstein" synonymous with "genius."
```

Type `exit` to quit the application.

## Swapping Models and Providers

You can choose which model provider to use (LM Studio or Ollama) and which specific model to run.

### Choosing a Provider

Use the `-provider` flag to specify which provider you want to use:

```bash
# Use LM Studio (default)
./inmemory -provider lmstudio

# Use Ollama
./inmemory -provider ollama
```

If you specify an unknown provider or don't specify a provider at all, WikiLLM will default to using LM Studio.

### LM Studio Models

When using LM Studio as your provider:

1. Open LM Studio and load your desired model
2. Start the local server in LM Studio (click "Start Server" in the interface)
3. Run WikiLLM with the model name that matches your loaded model:

```bash
./inmemory -provider lmstudio -model llama3
```

The model name should match what you've loaded in LM Studio. If you're using the default model in LM Studio, you can simply use the default model name:

```bash
./inmemory -provider lmstudio -model default
```

### Ollama Models

When using Ollama as your provider, you can specify various models including:

- llama2
- mistral
- vicuna
- orca-mini

For example:

```bash
./inmemory -provider ollama -model llama2
```

For a complete list of available Ollama models, visit the [Ollama Models Library](https://ollama.ai/library).

## Troubleshooting

### LM Studio Connection Issues

Make sure LM Studio is running and the server is started:

1. Open LM Studio
2. Load a model
3. Click "Start Server" in the interface
4. Ensure the server is running on the default port (1234)

### Ollama Connection Issues

Make sure Ollama is running before starting WikiLLM:

```bash
ollama serve
```

### Index Creation Failures

If you encounter issues during index creation:

1. Ensure you have enough disk space
2. Check that the Wikipedia dump file is valid XML
3. Try using a smaller Wikipedia dump for testing

### Memory Issues

Large Wikipedia dumps require significant memory. If you encounter memory issues:

1. Use a smaller Wikipedia dump
2. Increase your system's swap space
3. Run on a machine with more RAM

## License

This project is licensed under the MIT License - see the LICENSE file for details.
