# Troubleshooting Guide

## Common Issues

### GoReview CLI

#### "Ollama is not running"
**Problem:** The CLI cannot connect to the Ollama API.
**Solution:** Ensure Ollama is running (`ollama serve`) and accessible at the configured `base_url`.

#### "Model not found"
**Problem:** The requested model hasn't been pulled yet.
**Solution:** Run `ollama pull qwen2.5-coder:7b`.

### GitHub App

#### "Webhook signature mismatch"
**Problem:** `GITHUB_WEBHOOK_SECRET` in `.env` doesn't match the one in GitHub App settings.
**Solution:** Update the secret in both places.

#### "Private key not found"
**Problem:** The `.pem` file is missing or the path in `GITHUB_PRIVATE_KEY_PATH` is wrong.
**Solution:** Check the path and ensure the file is readable.

## Getting Help
If you encounter other issues, please open an issue in the repository.
