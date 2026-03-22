import { describe, it, expect, vi, beforeEach } from 'vitest';
import { ProbexClient, ProbexError } from '../src/client.js';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

function jsonResponse(data: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(data),
    text: () => Promise.resolve(JSON.stringify(data)),
  } as Response;
}

describe('ProbexClient', () => {
  let client: ProbexClient;

  beforeEach(() => {
    mockFetch.mockReset();
    client = new ProbexClient({ serverUrl: 'http://localhost:9712' });
  });

  it('health() calls GET /api/v1/health', async () => {
    const payload = { status: 'ok', version: '0.5.0' };
    mockFetch.mockResolvedValueOnce(jsonResponse(payload));

    const result = await client.health();

    expect(result).toEqual(payload);
    expect(mockFetch).toHaveBeenCalledWith(
      'http://localhost:9712/api/v1/health',
      expect.objectContaining({ method: 'GET' }),
    );
  });

  it('scan() calls POST /api/v1/scan', async () => {
    const profile = { id: 'p1', name: 'Test', base_url: 'http://api.example.com', endpoints: [] };
    mockFetch.mockResolvedValueOnce(jsonResponse(profile));

    const result = await client.scan({ base_url: 'http://api.example.com' });

    expect(result).toEqual(profile);
    expect(mockFetch).toHaveBeenCalledWith(
      'http://localhost:9712/api/v1/scan',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ base_url: 'http://api.example.com' }),
      }),
    );
  });

  it('run() calls POST /api/v1/run', async () => {
    const summary = {
      profile_id: 'p1',
      total_tests: 5,
      passed: 4,
      failed: 1,
      errors: 0,
      skipped: 0,
      duration: 1200,
      results: [],
      started_at: '2026-01-01T00:00:00Z',
      finished_at: '2026-01-01T00:00:01Z',
    };
    mockFetch.mockResolvedValueOnce(jsonResponse(summary));

    const result = await client.run({ categories: ['auth'] });

    expect(result).toEqual(summary);
    expect(mockFetch).toHaveBeenCalledWith(
      'http://localhost:9712/api/v1/run',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ categories: ['auth'] }),
      }),
    );
  });

  it('getProfile() calls GET /api/v1/profile', async () => {
    const profile = { id: 'p1', name: 'Test', base_url: 'http://api.example.com', endpoints: [] };
    mockFetch.mockResolvedValueOnce(jsonResponse(profile));

    const result = await client.getProfile();

    expect(result).toEqual(profile);
    expect(mockFetch).toHaveBeenCalledWith(
      'http://localhost:9712/api/v1/profile',
      expect.objectContaining({ method: 'GET' }),
    );
  });

  it('getResults() calls GET /api/v1/results', async () => {
    const summary = {
      profile_id: 'p1',
      total_tests: 3,
      passed: 3,
      failed: 0,
      errors: 0,
      skipped: 0,
      duration: 800,
      results: [],
      started_at: '2026-01-01T00:00:00Z',
      finished_at: '2026-01-01T00:00:01Z',
    };
    mockFetch.mockResolvedValueOnce(jsonResponse(summary));

    const result = await client.getResults();

    expect(result).toEqual(summary);
    expect(mockFetch).toHaveBeenCalledWith(
      'http://localhost:9712/api/v1/results',
      expect.objectContaining({ method: 'GET' }),
    );
  });

  it('throws ProbexError on non-2xx', async () => {
    mockFetch.mockResolvedValueOnce(jsonResponse({ error: 'not found' }, 404));

    await expect(client.getProfile()).rejects.toThrow(ProbexError);
    await expect(client.getProfile()).rejects.not.toThrow(); // reset - need fresh mock

    // Verify error properties
    mockFetch.mockResolvedValueOnce(jsonResponse({ error: 'server error' }, 500));
    try {
      await client.health();
      expect.unreachable('Should have thrown');
    } catch (err) {
      expect(err).toBeInstanceOf(ProbexError);
      expect((err as ProbexError).statusCode).toBe(500);
    }
  });
});
