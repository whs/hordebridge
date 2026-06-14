# Horde Bridge

Hordge Bridge is a bridge between AI providers' API and [AI Horde (Stable Horde)](https://stablehorde.net/contribute/joining) Worker API. It allows you to run AI Horde worker with your existing LLM setup.

Some possible use cases

- Bring your own inference engine (Ollama, Llama.cpp, vLLM, etc.) as AI Horde worker
- Using your own commercial LLM credential (eg. Fireworks, OpenAI, etc.) to provide free service to the AI Horde community
  - Note that AI Horde use **text** completion endpoint and not **chat** completion endpoint used in frontier models

I'm not responsible if you get banned from the LLM provider because someone on AI Horde submitted prompts that trigger guardrails.

## License



