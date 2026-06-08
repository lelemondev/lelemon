import process from 'node:process';
import type {
  NewAccessToken,
  NewAuthCode,
  NewClient,
  NewRefreshToken,
  OAuthStore,
  StoredAccessToken,
  StoredAuthCode,
  StoredClient,
  StoredRefreshToken,
} from '@mcify/core';

/**
 * `OAuthStore` backed by the Lelemon Go backend. The MCP authorization server (mcify) owns the
 * OAuth protocol logic but persists nothing locally — every store call is an RPC to
 * `POST {LELEMON_ENDPOINT}/api/v1/internal/oauth`, authenticated with the shared `MCP_STORE_SECRET`.
 *
 * The wire shapes are the mcify `OAuthStore` row types verbatim (the Go DTOs use the same camelCase
 * field names), so this adapter is a thin pass-through — the only translation is dates: outbound
 * `Date`s serialize to ISO strings via `JSON.stringify`, and inbound ISO strings are revived to
 * `Date` on the fields the interface types as `Date`.
 */
export interface HttpOAuthStoreOptions {
  /** Backend base URL. Defaults to LELEMON_ENDPOINT or https://api.lelemon.dev. */
  endpoint?: string;
  /** Shared service secret (MCP_STORE_SECRET). */
  secret: string;
}

const DEFAULT_ENDPOINT = 'https://api.lelemon.dev';

type Json = Record<string, unknown>;

function reviveClient(c: Json | null): StoredClient | null {
  if (!c) return null;
  return { ...(c as unknown as StoredClient), createdAt: new Date(c['createdAt'] as string) };
}

function reviveAuthCode(c: Json | null): StoredAuthCode | null {
  if (!c) return null;
  return { ...(c as unknown as StoredAuthCode), expiresAt: new Date(c['expiresAt'] as string) };
}

function reviveAccess(t: Json | null): StoredAccessToken | null {
  if (!t) return null;
  const revoked = t['revokedAt'];
  return {
    ...(t as unknown as StoredAccessToken),
    expiresAt: new Date(t['expiresAt'] as string),
    revokedAt: revoked ? new Date(revoked as string) : null,
  };
}

function reviveRefresh(t: Json | null): StoredRefreshToken | null {
  if (!t) return null;
  const consumed = t['consumedAt'];
  return {
    ...(t as unknown as StoredRefreshToken),
    expiresAt: new Date(t['expiresAt'] as string),
    consumedAt: consumed ? new Date(consumed as string) : null,
  };
}

export class HttpOAuthStore implements OAuthStore {
  private readonly url: string;
  private readonly secret: string;

  constructor(options: HttpOAuthStoreOptions) {
    // The secret may be empty at construction (e.g. during `mcify build` with no env); it is
    // required only when a call is actually made — see `call()`.
    const base = options.endpoint ?? process.env['LELEMON_ENDPOINT'] ?? DEFAULT_ENDPOINT;
    this.url = `${base.replace(/\/+$/, '')}/api/v1/internal/oauth`;
    this.secret = options.secret;
  }

  private async call(op: string, payload: Json = {}): Promise<Json> {
    if (!this.secret) {
      throw new Error('MCP_STORE_SECRET is not configured — the MCP cannot reach the OAuth store.');
    }
    const res = await fetch(this.url, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${this.secret}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ op, ...payload }),
    });
    if (!res.ok) {
      const body = await res.text().catch(() => '');
      throw new Error(`OAuth store op "${op}" failed (${res.status}): ${body.slice(0, 200)}`);
    }
    return (await res.json()) as Json;
  }

  async insertClient(client: NewClient): Promise<StoredClient> {
    await this.call('insertClient', { client });
    // The backend returns {} on insert; re-read to get the canonical stored row (with createdAt).
    const stored = await this.getClientById(client.clientId);
    if (!stored) throw new Error('insertClient: client not found after insert');
    return stored;
  }

  async getClientById(clientId: string): Promise<StoredClient | null> {
    const out = await this.call('getClientById', { clientId });
    return reviveClient(out['client'] as Json | null);
  }

  async findClientsByName(clientName: string | null): Promise<StoredClient[]> {
    const out = await this.call('findClientsByName', { clientName: clientName ?? '' });
    const clients = (out['clients'] as Json[] | null) ?? [];
    return clients.map((c) => reviveClient(c)).filter((c): c is StoredClient => c !== null);
  }

  async insertAuthorizationCode(code: NewAuthCode): Promise<void> {
    await this.call('insertAuthCode', { code });
  }

  async consumeAuthorizationCode(codeHash: string): Promise<StoredAuthCode | null> {
    const out = await this.call('consumeAuthCode', { codeHash });
    return reviveAuthCode(out['code'] as Json | null);
  }

  async insertAccessToken(token: NewAccessToken): Promise<void> {
    await this.call('insertAccessToken', { accessToken: token });
  }

  async getAccessTokenByHash(tokenHash: string): Promise<StoredAccessToken | null> {
    const out = await this.call('getAccessTokenByHash', { tokenHash });
    return reviveAccess(out['token'] as Json | null);
  }

  async insertRefreshToken(token: NewRefreshToken): Promise<StoredRefreshToken> {
    const out = await this.call('insertRefreshToken', { refreshToken: token });
    return {
      id: out['id'] as string,
      clientId: token.clientId,
      subjectKey: token.subjectKey,
      scope: token.scope,
      expiresAt: token.expiresAt,
      consumedAt: null,
    };
  }

  async getRefreshTokenByHash(tokenHash: string): Promise<StoredRefreshToken | null> {
    const out = await this.call('getRefreshTokenByHash', { tokenHash });
    return reviveRefresh(out['token'] as Json | null);
  }

  async consumeRefreshToken(id: string): Promise<boolean> {
    const out = await this.call('consumeRefreshToken', { id });
    return out['consumed'] === true;
  }

  async setRefreshRotatedTo(id: string, rotatedToId: string): Promise<void> {
    await this.call('setRefreshRotatedTo', { id, rotatedToId });
  }

  async revokeChain(subjectKey: string, clientId: string): Promise<void> {
    await this.call('revokeChain', { subjectKey, clientId });
  }
}
