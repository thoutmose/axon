// Swagger UI for the live OpenAPI document. "Try it out" sends requests as
// the signed-in browser session, like the app does.
window.SwaggerUIBundle({
	url: '/api/v1/openapi.yaml',
	dom_id: '#swagger-ui',
	deepLinking: true,
	docExpansion: 'none',
	tryItOutEnabled: false
});
