import * as vscode from 'vscode';
import { ProbexClient } from './client';

export class ResultsTreeProvider implements vscode.TreeDataProvider<ResultItem> {
    private _onDidChangeTreeData = new vscode.EventEmitter<ResultItem | undefined>();
    readonly onDidChangeTreeData = this._onDidChangeTreeData.event;

    private results: any[] = [];
    private summary: any = null;

    constructor(private client: ProbexClient) {}

    refresh(): void {
        this.loadResults();
    }

    private async loadResults(): Promise<void> {
        try {
            const data = await this.client.getResults();
            this.summary = data;
            this.results = data.results || [];
        } catch {
            this.summary = null;
            this.results = [];
        }
        this._onDidChangeTreeData.fire(undefined);
    }

    getTreeItem(element: ResultItem): vscode.TreeItem {
        return element;
    }

    getChildren(element?: ResultItem): ResultItem[] {
        if (!element) {
            // Root level — show summary + results.
            if (!this.summary) {
                return [new ResultItem('No results yet — run tests first', 'info')];
            }

            const items: ResultItem[] = [];
            items.push(new ResultItem(
                `${this.summary.total_tests} tests: ${this.summary.passed} passed, ${this.summary.failed} failed`,
                this.summary.failed > 0 ? 'warning' : 'success'
            ));

            // Add individual results.
            for (const r of this.results) {
                items.push(new ResultItem(
                    r.test_name,
                    r.status === 'passed' ? 'passed' : r.status === 'failed' ? 'failed' : 'error',
                    r
                ));
            }

            return items;
        }

        return [];
    }
}

class ResultItem extends vscode.TreeItem {
    constructor(
        label: string,
        public readonly resultType: string,
        public readonly result?: any,
    ) {
        super(label, vscode.TreeItemCollapsibleState.None);

        if (result) {
            this.tooltip = `${result.test_name}\nStatus: ${result.status}\nSeverity: ${result.severity}\nCategory: ${result.category}`;
            this.description = `${result.severity} | ${result.category}`;
        }

        switch (resultType) {
            case 'passed':
                this.iconPath = new vscode.ThemeIcon('check', new vscode.ThemeColor('testing.iconPassed'));
                break;
            case 'failed':
                this.iconPath = new vscode.ThemeIcon('close', new vscode.ThemeColor('testing.iconFailed'));
                break;
            case 'error':
                this.iconPath = new vscode.ThemeIcon('warning', new vscode.ThemeColor('list.warningForeground'));
                break;
            case 'success':
                this.iconPath = new vscode.ThemeIcon('pass-filled', new vscode.ThemeColor('testing.iconPassed'));
                break;
            case 'warning':
                this.iconPath = new vscode.ThemeIcon('warning', new vscode.ThemeColor('list.warningForeground'));
                break;
            default:
                this.iconPath = new vscode.ThemeIcon('info');
        }
    }
}
