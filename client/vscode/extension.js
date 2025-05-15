// @ts-check
const { LanguageClient, TransportKind } = require("vscode-languageclient/node");
const path = require("path");

let client = null;

module.exports = {
	/** @param {import("vscode").ExtensionContext} context*/
	activate(context) {
		const serverModule = context.asAbsolutePath(
			path.join("dist", "evergreenlsp"),
		);
		/** @type {import("vscode-languageclient/node").ServerOptions} */
		const serverOptions = {
			run: {
				command: serverModule,
				args: ["lsp"],
				transport: TransportKind.stdio,
			},
			debug: {
				args: ["lsp", "-v"],
				command: serverModule,
				transport: TransportKind.stdio,
			},
		};

		/** @type {import("vscode-languageclient/node").LanguageClientOptions} */
		const clientOptions = {
			documentSelector: [{ scheme: "file", language: "yaml" }],
			markdown: { isTrusted: true },
		};

		client = new LanguageClient(
			"evergreen-lsp",
			"Evergreen CI Language Server",
			serverOptions,
			clientOptions,
		);

		client.start();
	},
	deactivate() {
		if (!client) {
			return undefined;
		}
		return client.stop();
	},
};
