import dotenv from 'dotenv';
import { z } from 'zod';

dotenv.config();

const envSchema = z.object({
  GITHUB_APP_ID: z.string(),
  GITHUB_WEBHOOK_SECRET: z.string(),
  GITHUB_PRIVATE_KEY_PATH: z.string(),
  PORT: z.string().default('3000'),
  NODE_ENV: z.enum(['development', 'production', 'test']).default('development'),
  LOG_LEVEL: z.enum(['fatal', 'error', 'warn', 'info', 'debug', 'trace']).default('info'),
  OLLAMA_HOST: z.string().default('http://localhost:11434'),
  OLLAMA_MODEL: z.string().default('qwen2.5-coder:7b'),
});

export type Config = z.infer<typeof envSchema>;

const env = envSchema.safeParse(process.env);

if (!env.success) {
  console.error('❌ Invalid environment variables:', env.error.format());
  // Don't exit process here for testing purposes, but in real runtime it might crash later
  // or we can throw
  if (process.env.NODE_ENV !== 'test') {
    process.exit(1);
  }
}

export const config = env.success ? env.data : (process.env as unknown as Config);
