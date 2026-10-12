// Copies Swagger UI (Apache-2.0) from node_modules into static/api-docs, so
// the API reference is served by Axon itself and never loads a CDN.
import { copyFileSync, mkdirSync } from 'node:fs';

const from = 'node_modules/swagger-ui-dist';
const to = 'static/api-docs/vendor';
mkdirSync(to, { recursive: true });
for (const file of ['swagger-ui-bundle.js', 'swagger-ui.css', 'LICENSE', 'NOTICE']) {
	copyFileSync(`${from}/${file}`, `${to}/${file}`);
}
