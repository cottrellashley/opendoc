---
title: "Settings Guide"
---

# Settings Guide

The workbench settings panel lets you connect AI providers, manage API keys, and configure the chatbot. Click the **gear icon** in the top-right corner of the workbench to open it.

---

## AI Providers

The chatbot supports three AI providers. You can switch between them at any time using the provider buttons above the chat input.

| Provider | How to connect | Models available |
|----------|---------------|-----------------|
| **GitHub Copilot** | Sign in with GitHub (SSO) | GPT-4.1, Claude Sonnet 4, GPT-4o, Gemini 2.5 Pro |
| **Anthropic** | Paste API key | Claude Sonnet 4, Claude Haiku 3.5 |
| **OpenAI** | Paste API key | GPT-4o, GPT-4o Mini, o3 Mini |

---

## GitHub Copilot (recommended)

If you have a **GitHub Copilot subscription** (individual, business, or enterprise), you can use it directly — no API key needed.

### How to connect

1. Open **Settings** (gear icon, top-right)
2. Find the **GitHub Copilot** card
3. Click **Sign in with GitHub**
4. You will see a **device code** and a link to GitHub
5. Click the link (or go to [github.com/login/device](https://github.com/login/device))
6. Paste the code and authorize
7. The workbench will detect the authorization automatically

Once connected, select **GitHub Copilot** from the provider buttons in the chat panel. You can choose from multiple models including GPT-4.1, Claude Sonnet 4, GPT-4o, and Gemini 2.5 Pro.

:::sidenote What is the device code flow?
This is the same secure OAuth flow used by GitHub CLI, VS Code, and other developer tools. OpenDoc never sees your GitHub password — it only receives a scoped token for Copilot access.
:::

### Disconnecting

To disconnect Copilot, go to Settings and click **Disconnect** on the Copilot card. You can also revoke access from your [GitHub settings](https://github.com/settings/apps/authorizations).

---

## Anthropic (Claude)

To use Claude models directly:

1. Go to [console.anthropic.com](https://console.anthropic.com/) and create an account
2. Navigate to **API Keys** and create a new key
3. Open **Settings** in the workbench
4. Paste your API key in the **Anthropic** field and click **Save**

Alternatively, set the environment variable before starting the workbench:

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
opendoc workbench
```

---

## OpenAI (GPT)

To use GPT models directly:

1. Go to [platform.openai.com](https://platform.openai.com/) and create an account
2. Navigate to **API Keys** and create a new key
3. Open **Settings** in the workbench
4. Paste your API key in the **OpenAI** field and click **Save**

Or set the environment variable:

```bash
export OPENAI_API_KEY="sk-..."
opendoc workbench
```

---

## Switching providers and models

Once you have at least one provider connected, you will see **provider buttons** above the chat input:

- Click a provider name to switch (e.g. **GitHub Copilot**, **Claude**, **GPT**)
- Use the **model dropdown** next to the buttons to pick a specific model
- Switching providers starts a new chat session

:::sidenote Which provider should I use?
**GitHub Copilot** is the easiest to set up if you already have a subscription — no API keys, no billing. It also gives you access to models from multiple vendors (OpenAI, Anthropic, Google) through a single connection. If you do not have Copilot, Anthropic's Claude and OpenAI's GPT are both excellent choices.
:::

---

## API key storage

API keys are stored locally in `~/.config/opendoc/secrets.yml` on your machine. They are **never sent anywhere** except to the provider's own API endpoint.

| Method | Location | Priority |
|--------|----------|----------|
| Environment variable | `ANTHROPIC_API_KEY`, `OPENAI_API_KEY` | Highest |
| Settings file | `~/.config/opendoc/secrets.yml` | Fallback |
| GitHub Copilot | OAuth token (auto-managed) | Separate flow |

Environment variables take priority over the settings file, so you can override keys per-session without changing your saved config.

---

## Troubleshooting

**"API key not set" error when chatting**
- Open Settings and check that your key is saved, or that Copilot shows as connected
- If you just connected Copilot, try refreshing the page

**Copilot sign-in stuck on "Waiting for authorization"**
- Make sure you completed the device code flow on GitHub
- Check that your GitHub account has an active Copilot subscription
- Try disconnecting and signing in again

**Want to use a different model?**
- Use the model dropdown next to the provider buttons in the chat panel
- Or set an environment variable: `ANTHROPIC_MODEL`, `OPENAI_MODEL`, or `COPILOT_MODEL`
