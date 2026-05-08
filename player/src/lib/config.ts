export type Config = {
  grpcGatewayUrl: string;
  iamBaseUrl: string;
  discordClientId: string;
  // Optional Discord-server invite URL surfaced on the Pending page so
  // applicants can join the studio's Discord while waiting for approval.
  // Required for outbound DMs to land — Discord blocks bot DMs when the
  // bot and recipient share no guild (error 50278). See the runbook
  // docs/runbooks/setup-ags-discord.md § "Discord bot + server".
  discordInviteUrl?: string;
};

export class ConfigError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'ConfigError';
  }
}

const URL_KEYS = ['grpcGatewayUrl', 'iamBaseUrl'] as const;
const REQUIRED_KEYS = [...URL_KEYS, 'discordClientId'] as const;

export function parseConfig(raw: string): Config {
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch (err) {
    throw new ConfigError(`config.json is not valid JSON: ${(err as Error).message}`);
  }

  if (parsed === null || typeof parsed !== 'object' || Array.isArray(parsed)) {
    throw new ConfigError('config.json must be a JSON object');
  }

  const obj = parsed as Record<string, unknown>;

  for (const key of REQUIRED_KEYS) {
    if (!(key in obj)) {
      throw new ConfigError(`config.json missing required key: ${key}`);
    }
    if (typeof obj[key] !== 'string') {
      throw new ConfigError(`config.json key ${key} must be a string`);
    }
  }

  for (const key of URL_KEYS) {
    const value = obj[key] as string;
    try {
      // eslint-disable-next-line no-new
      new URL(value);
    } catch {
      throw new ConfigError(`config.json ${key} is not a valid URL: ${value}`);
    }
  }

  const discordClientId = obj.discordClientId as string;
  if (discordClientId.length === 0) {
    throw new ConfigError('config.json discordClientId must not be empty');
  }

  let discordInviteUrl: string | undefined;
  if ('discordInviteUrl' in obj && obj.discordInviteUrl !== '' && obj.discordInviteUrl != null) {
    if (typeof obj.discordInviteUrl !== 'string') {
      throw new ConfigError('config.json key discordInviteUrl must be a string');
    }
    try {
      // eslint-disable-next-line no-new
      new URL(obj.discordInviteUrl);
    } catch {
      throw new ConfigError(
        `config.json discordInviteUrl is not a valid URL: ${obj.discordInviteUrl}`,
      );
    }
    discordInviteUrl = obj.discordInviteUrl;
  }

  return {
    grpcGatewayUrl: obj.grpcGatewayUrl as string,
    iamBaseUrl: obj.iamBaseUrl as string,
    discordClientId,
    ...(discordInviteUrl !== undefined ? { discordInviteUrl } : {}),
  };
}

export async function loadConfig(url: string = '/config.json'): Promise<Config> {
  const res = await fetch(url, { cache: 'no-cache' });
  if (!res.ok) {
    throw new ConfigError(`config.json fetch failed: ${res.status} ${res.statusText}`);
  }
  const raw = await res.text();
  return parseConfig(raw);
}
