/**
 * HTTP client for communicating with the PROBEX REST API server.
 */
export class ProbexClient {
    private baseUrl: string;

    constructor(baseUrl: string) {
        this.baseUrl = baseUrl.replace(/\/$/, '');
    }

    async health(): Promise<{ status: string; version: string }> {
        return this.get('/api/v1/health');
    }

    async scan(targetUrl: string, maxDepth = 3): Promise<any> {
        return this.post('/api/v1/scan', {
            base_url: targetUrl,
            max_depth: maxDepth,
            concurrency: 10,
        });
    }

    async run(categories?: string[]): Promise<any> {
        const body: any = {};
        if (categories && categories.length > 0) {
            body.categories = categories;
        }
        return this.post('/api/v1/run', body);
    }

    async getProfile(): Promise<any> {
        return this.get('/api/v1/profile');
    }

    async getResults(): Promise<any> {
        return this.get('/api/v1/results');
    }

    private async get(path: string): Promise<any> {
        const resp = await fetch(`${this.baseUrl}${path}`);
        if (!resp.ok) {
            const body = await resp.text();
            throw new Error(`HTTP ${resp.status}: ${body}`);
        }
        return resp.json();
    }

    private async post(path: string, body: any): Promise<any> {
        const resp = await fetch(`${this.baseUrl}${path}`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body),
        });
        if (!resp.ok) {
            const text = await resp.text();
            throw new Error(`HTTP ${resp.status}: ${text}`);
        }
        return resp.json();
    }
}
