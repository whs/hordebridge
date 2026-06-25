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

## Responses API

Hordebridge supports detection and conversion of chat templates to [OpenResponses API](https://www.openresponses.org/).

The supported chat template tags are:

- KoboldCPP (`{{[SYSTEM]}}, {{[INPUT]}}, {{[OUTPUT]}}`)
- Alpaca (`### Instruction:\n, ### Response:\n`)

Continuation is supported as prefills - the last turn is tagged as "assistant" and the agent is supposed to continue writing
that turn without creating double assistant turns in a row.
You'll need to ensure that the underlying API is supported.

## Content moderation

Hordebridge supports content moderation. It is off by default. Use `--classifier-block-nsfw` and/or `--classifier-block-csam` to enable.
Note that `--classifier-block-nsfw` is not allowed together with `--no-nsfw`.

The classifier works by sending the input prompt to your current model, along with a system prompt.
The model must support tool calls for this to work. Alternatively, you can configure alternative OpenResponses server/model as well.

When the classifier is active (by enable *any* block option), the following actions will be taken:

- If CSAM is detected, the job state will be reported as "csam" (even if block-csam is off) and if the block is enabled, the generated text replaced with a blocked message.
- If NSFW content is detected and the block is on, the job state is reported as censored and the generated text replaced with a blocked message.

If the classifier errors out, no action will be taken unless `--classifier-fail-close` is set.

## License

[MIT License](https://spdx.org/licenses/MIT.html)
