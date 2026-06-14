# Horde Bridge

Horde Bridge is a bridge between AI providers' API and [AI Horde (Stable Horde)](https://stablehorde.net/contribute/joining) Worker API. It allows you to run AI Horde worker with your existing LLM setup.

Some possible use cases

- Bring your own inference engine (Ollama, Llama.cpp, vLLM, etc.) as AI Horde worker
- Using your own commercial LLM credential (eg. Fireworks, OpenAI, etc.) to provide free service to the AI Horde community
  - Note that AI Horde use **text** completion endpoint and not **chat** completion endpoint used in frontier models

I'm not responsible if you get banned from the LLM provider because someone on AI Horde submitted prompts that trigger guardrails.

## Usage

1. Build this from source by `go install github.com/whs/hordebridge@latest`
2. Run `hordebridge --horde-api-key=... --horde-model=... --max-context-length 131000 --max-length 131000 --openai-server=... --openai-api-key=... --openai-model=... --worker-name=yourname`
   - `--horde-model` is the model name appearing on AI Horde website. See https://aihorde.net/details/models/text for list.
   - `--openai-model` is the model name to pass to OpenAI-compatible text generation endpoint 
   - `--max-context-length` is the max context length of the model in tokens
   - `--max-length` is the max input length of the model in tokens
   - `--help` for additional options

All command line arguments can be specified as environment variables. For example, --horde-api-key can be set as HORDE_API_KEY

## License

[MIT License](https://spdx.org/licenses/MIT.html)
