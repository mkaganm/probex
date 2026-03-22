import * as vscode from 'vscode';
import { ProbexClient } from './client';

export class EndpointTreeProvider implements vscode.TreeDataProvider<EndpointItem> {
    private _onDidChangeTreeData = new vscode.EventEmitter<EndpointItem | undefined>();
    readonly onDidChangeTreeData = this._onDidChangeTreeData.event;

    private endpoints: any[] = [];

    constructor(private client: ProbexClient) {}

    refresh(): void {
        this.loadEndpoints();
    }

    private async loadEndpoints(): Promise<void> {
        try {
            const profile = await this.client.getProfile();
            this.endpoints = profile.endpoints || [];
        } catch {
            this.endpoints = [];
        }
        this._onDidChangeTreeData.fire(undefined);
    }

    getTreeItem(element: EndpointItem): vscode.TreeItem {
        return element;
    }

    getChildren(): EndpointItem[] {
        return this.endpoints.map(ep => {
            const label = `${ep.method} ${ep.path}`;
            const item = new EndpointItem(label, ep);
            return item;
        });
    }
}

class EndpointItem extends vscode.TreeItem {
    constructor(label: string, public readonly endpoint: any) {
        super(label, vscode.TreeItemCollapsibleState.None);

        this.tooltip = `${endpoint.method} ${endpoint.base_url}${endpoint.path}`;
        this.description = endpoint.source || '';

        // Icon based on method.
        const method = endpoint.method?.toUpperCase() || '';
        if (method === 'GET') {
            this.iconPath = new vscode.ThemeIcon('arrow-down', new vscode.ThemeColor('charts.blue'));
        } else if (method === 'POST') {
            this.iconPath = new vscode.ThemeIcon('add', new vscode.ThemeColor('charts.green'));
        } else if (method === 'PUT' || method === 'PATCH') {
            this.iconPath = new vscode.ThemeIcon('edit', new vscode.ThemeColor('charts.yellow'));
        } else if (method === 'DELETE') {
            this.iconPath = new vscode.ThemeIcon('trash', new vscode.ThemeColor('charts.red'));
        } else if (method === 'QUERY' || method === 'MUTATION') {
            this.iconPath = new vscode.ThemeIcon('symbol-method', new vscode.ThemeColor('charts.purple'));
        } else if (method === 'WS') {
            this.iconPath = new vscode.ThemeIcon('broadcast', new vscode.ThemeColor('charts.green'));
        } else if (method === 'GRPC') {
            this.iconPath = new vscode.ThemeIcon('server', new vscode.ThemeColor('charts.orange'));
        }
    }
}
