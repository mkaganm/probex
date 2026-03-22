import * as vscode from 'vscode';
import { ProbexClient } from './client';
import { EndpointTreeProvider } from './endpoints';
import { ResultsTreeProvider } from './results';

let outputChannel: vscode.OutputChannel;
let client: ProbexClient;
let endpointProvider: EndpointTreeProvider;
let resultsProvider: ResultsTreeProvider;

export function activate(context: vscode.ExtensionContext) {
    outputChannel = vscode.window.createOutputChannel('PROBEX');

    const config = vscode.workspace.getConfiguration('probex');
    const serverUrl = config.get<string>('serverUrl', 'http://localhost:9712');
    client = new ProbexClient(serverUrl);

    // Tree data providers.
    endpointProvider = new EndpointTreeProvider(client);
    resultsProvider = new ResultsTreeProvider(client);

    vscode.window.registerTreeDataProvider('probex.endpoints', endpointProvider);
    vscode.window.registerTreeDataProvider('probex.results', resultsProvider);

    // Register commands.
    context.subscriptions.push(
        vscode.commands.registerCommand('probex.scan', handleScan),
        vscode.commands.registerCommand('probex.run', handleRun),
        vscode.commands.registerCommand('probex.watch', handleWatch),
        vscode.commands.registerCommand('probex.dashboard', handleDashboard),
        vscode.commands.registerCommand('probex.init', handleInit),
        vscode.commands.registerCommand('probex.graph', handleGraph),
    );

    outputChannel.appendLine('PROBEX extension activated');

    // Auto-scan if configured.
    if (config.get<boolean>('autoScan', false)) {
        vscode.commands.executeCommand('probex.scan');
    }
}

async function handleScan() {
    const targetUrl = await vscode.window.showInputBox({
        prompt: 'Enter API base URL to scan',
        placeHolder: 'http://localhost:8080',
        value: 'http://localhost:8080',
    });

    if (!targetUrl) return;

    await vscode.window.withProgress({
        location: vscode.ProgressLocation.Notification,
        title: 'PROBEX: Scanning API...',
        cancellable: false,
    }, async () => {
        try {
            const result = await client.scan(targetUrl);
            const count = result.endpoints?.length ?? 0;
            vscode.window.showInformationMessage(
                `PROBEX: Discovered ${count} endpoints`
            );
            outputChannel.appendLine(`Scan complete: ${count} endpoints at ${targetUrl}`);
            endpointProvider.refresh();
        } catch (err: any) {
            vscode.window.showErrorMessage(`PROBEX scan failed: ${err.message}`);
            outputChannel.appendLine(`Scan error: ${err.message}`);
        }
    });
}

async function handleRun() {
    await vscode.window.withProgress({
        location: vscode.ProgressLocation.Notification,
        title: 'PROBEX: Running tests...',
        cancellable: false,
    }, async () => {
        try {
            const result = await client.run();
            const msg = `${result.total_tests} tests: ${result.passed} passed, ${result.failed} failed, ${result.errors} errors`;

            if (result.failed > 0 || result.errors > 0) {
                vscode.window.showWarningMessage(`PROBEX: ${msg}`);
            } else {
                vscode.window.showInformationMessage(`PROBEX: ${msg}`);
            }

            outputChannel.appendLine(`Run complete: ${msg}`);
            resultsProvider.refresh();
        } catch (err: any) {
            vscode.window.showErrorMessage(`PROBEX run failed: ${err.message}`);
            outputChannel.appendLine(`Run error: ${err.message}`);
        }
    });
}

async function handleWatch() {
    const config = vscode.workspace.getConfiguration('probex');
    const binaryPath = config.get<string>('binaryPath', 'probex');

    const terminal = vscode.window.createTerminal({
        name: 'PROBEX Watch',
        shellPath: binaryPath,
        shellArgs: ['watch'],
    });
    terminal.show();
}

async function handleDashboard() {
    const config = vscode.workspace.getConfiguration('probex');
    const serverUrl = config.get<string>('serverUrl', 'http://localhost:9712');

    const dashboardUrl = `${serverUrl}/dashboard`;
    vscode.env.openExternal(vscode.Uri.parse(dashboardUrl));
}

async function handleInit() {
    const config = vscode.workspace.getConfiguration('probex');
    const binaryPath = config.get<string>('binaryPath', 'probex');

    const terminal = vscode.window.createTerminal('PROBEX Init');
    terminal.sendText(`${binaryPath} config init`);
    terminal.show();
}

async function handleGraph() {
    try {
        const profile = await client.getProfile();
        if (!profile || !profile.endpoints) {
            vscode.window.showWarningMessage('PROBEX: No profile found. Run a scan first.');
            return;
        }

        const panel = vscode.window.createWebviewPanel(
            'probexGraph',
            'PROBEX: Endpoint Graph',
            vscode.ViewColumn.One,
            { enableScripts: true }
        );

        panel.webview.html = buildGraphHTML(profile.endpoints);
    } catch (err: any) {
        vscode.window.showErrorMessage(`PROBEX: ${err.message}`);
    }
}

function buildGraphHTML(endpoints: any[]): string {
    const nodes = endpoints.map((ep: any, i: number) => ({
        id: i,
        label: `${ep.method} ${ep.path}`,
        method: ep.method,
    }));

    return `<!DOCTYPE html>
<html>
<head>
<style>
body { background: #0d1117; color: #c9d1d9; font-family: monospace; padding: 20px; }
h2 { color: #00FF88; margin-bottom: 16px; }
.endpoint { padding: 6px 12px; margin: 4px 0; border-radius: 6px; background: #161b22; border: 1px solid #30363d; }
.GET { border-left: 3px solid #00D4FF; }
.POST { border-left: 3px solid #00FF88; }
.PUT, .PATCH { border-left: 3px solid #FFD700; }
.DELETE { border-left: 3px solid #FF4444; }
.QUERY { border-left: 3px solid #BB88FF; }
.MUTATION { border-left: 3px solid #FF88BB; }
.WS { border-left: 3px solid #88FFBB; }
.GRPC { border-left: 3px solid #FFBB88; }
.method { font-weight: bold; margin-right: 8px; }
</style>
</head>
<body>
<h2>PROBEX — Endpoint Graph</h2>
<p>${nodes.length} endpoints discovered</p>
<div style="margin-top: 16px;">
${nodes.map((n: any) => `<div class="endpoint ${n.method}"><span class="method">${n.label}</span></div>`).join('\n')}
</div>
</body>
</html>`;
}

export function deactivate() {
    outputChannel?.dispose();
}
